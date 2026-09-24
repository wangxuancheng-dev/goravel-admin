package admin

import (
	"fmt"
	appfacades "goravel/app/facades"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/coder/websocket"
	apphttp "github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"
	"github.com/goravel/framework/support/str"
	"github.com/oklog/ulid/v2"

	apperrors "goravel/app/errors"
	"goravel/app/http/helpers"
	appmiddleware "goravel/app/http/middleware"
	"goravel/app/http/response"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/app/tenancy"
	"goravel/app/utils/logger"
	wsnotifications "goravel/app/websocket/notifications"
)

type NotificationWsController struct{}

const (
	wsTicketCachePrefix = "ws:ticket:"
	wsTicketTTL         = 60 * time.Second
)

func NewNotificationWsController() *NotificationWsController {
	return &NotificationWsController{}
}

func (r *NotificationWsController) tokenService(ctx apphttp.Context) services.TokenService {
	return services.NewTokenServiceImpl(ctx)
}

func (r *NotificationWsController) Ticket(ctx apphttp.Context) apphttp.Response {
	admin := r.currentAdmin(ctx)
	if admin == nil {
		return response.Error(ctx, http.StatusUnauthorized, "not_logged_in")
	}

	token := str.Of(ctx.Request().Header("Authorization", "")).ChopStart("Bearer ").Trim().String()
	if token == "" {
		return response.Error(ctx, http.StatusUnauthorized, "token_required")
	}

	ticket := strings.ToLower(ulid.Make().String())
	// Ticket keys stay unprefixed: WS upgrade has no Tenant middleware, but ULID is unique.
	// Embed tenant id so Server can BindHTTP before token/admin lookup.
	tenantID, _ := helpers.GetTenantIDFromContext(ctx)
	if tenancy.Enabled() && tenantID == 0 {
		logger.WarnfHTTP(ctx, "WebSocket ticket refused: tenant not bound (send X-Tenant-ID when calling api.* host)")
		return response.Error(ctx, http.StatusBadRequest, "tenant_required")
	}
	cacheKey := wsTicketCachePrefix + ticket
	// Format: tenantID|adminID|originHost|token (token last so it may contain "|").
	originHost := clientPageHost(ctx)
	cacheValue := fmt.Sprintf("%d|%d|%s|%s", tenantID, admin.ID, originHost, token)
	if err := facades.Cache().Put(cacheKey, cacheValue, wsTicketTTL); err != nil {
		return response.ErrorWithLog(ctx, "notification", err, map[string]any{
			"admin_id": admin.ID,
		})
	}

	logger.InfofHTTP(ctx, "WebSocket ticket issued tenant_id=%d origin_host=%s", tenantID, originHost)

	return response.Success(ctx, apphttp.Json{
		"ticket":     ticket,
		"expires_in": int(wsTicketTTL / time.Second),
	})
}

func (r *NotificationWsController) Server(ctx apphttp.Context) apphttp.Response {
	logger.InfofHTTP(ctx, "WebSocket connection attempt path=%s upgrade=%s connection=%s origin=%s",
		ctx.Request().Path(),
		ctx.Request().Header("Upgrade", ""),
		ctx.Request().Header("Connection", ""),
		ctx.Request().Header("Origin", ""))

	token, fromTicket, ticketOriginHost := r.extractToken(ctx)
	if token == "" {
		logger.WarnfHTTP(ctx, "WebSocket connection rejected: token_required")
		// Must return Error: discarded Error + Abort() defaults to empty 400 body.
		return response.Error(ctx, http.StatusUnauthorized, "token_required")
	}

	// Non-ticket Authorization path: bind tenant from header/query if still unbound.
	if tenancy.Enabled() {
		if _, ok := helpers.GetTenantIDFromContext(ctx); !ok {
			if err := services.NewTenantConnectionService().BindHTTP(ctx, ""); err != nil {
				logger.WarnfHTTP(ctx, "WebSocket tenant bind failed: %v", err)
				if businessErr, ok := apperrors.GetBusinessError(err); ok {
					return response.Error(ctx, http.StatusBadRequest, businessErr.Code)
				}
				return response.Error(ctx, http.StatusBadRequest, "tenant_required")
			}
		}
	}

	token = str.Of(token).ChopStart("Bearer ").Trim().String()
	accessToken, err := r.tokenService(ctx).FindToken(token)
	if err != nil || accessToken == nil || accessToken.TokenableType != "admin" {
		logger.WarnfHTTP(ctx, "WebSocket connection rejected: invalid_token")
		return response.Error(ctx, http.StatusUnauthorized, "invalid_token")
	}

	var admin models.Admin
	if err := appfacades.OrmQuery(ctx).Where("id", accessToken.TokenableID).FirstOrFail(&admin); err != nil {
		logger.WarnfHTTP(ctx, "WebSocket connection rejected: user_not_found")
		return response.Error(ctx, http.StatusUnauthorized, "user_not_found")
	}
	_ = r.tokenService(ctx).UpdateLastUsedAt(token)

	req := ctx.Request().Origin()
	tenantID, _ := helpers.GetTenantIDFromContext(ctx)
	if !r.isOriginAllowed(req, tenantID, ticketOriginHost, fromTicket) {
		logger.WarnfHTTP(ctx, "WebSocket connection rejected: origin_not_allowed origin=%s host=%s tenant_id=%d ticket_origin=%s from_ticket=%v",
			req.Header.Get("Origin"), req.Host, tenantID, ticketOriginHost, fromTicket)
		return response.Error(ctx, http.StatusForbidden, "origin_not_allowed")
	}

	// Origin already verified above; skip library check.
	conn, err := websocket.Accept(ctx.Response().Writer(), req, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		logger.ErrorfHTTP(ctx, "notification ws upgrade error: %v", err)
		// Accept may have already written a status; still return a body when possible.
		return ctx.Response().String(http.StatusInternalServerError, "upgrade_failed: "+err.Error())
	}

	wsnotifications.Hub().RegisterConnection(conn, tenantID, admin.ID)
	logger.InfofHTTP(ctx, "WebSocket connected admin_id=%d tenant_id=%d origin=%s from_ticket=%v",
		admin.ID, tenantID, req.Header.Get("Origin"), fromTicket)

	return nil
}

func (r *NotificationWsController) extractToken(ctx apphttp.Context) (token string, fromTicket bool, ticketOriginHost string) {
	ticket := str.Of(ctx.Request().Query("ticket")).Trim().String()
	if ticket != "" {
		tok, originHost, ok := r.consumeTicket(ctx, ticket)
		if ok {
			return tok, true, originHost
		}
		// Ticket present but unusable: do not fall through to Authorization (browsers omit it).
		logger.WarnfHTTP(ctx, "WebSocket ticket invalid, expired, or tenant bind failed")
		return "", false, ""
	}

	// Keep header parsing only for non-browser clients.
	authorization := str.Of(ctx.Request().Header("Authorization", "")).Trim().String()
	if authorization != "" {
		return authorization, false, ""
	}

	return "", false, ""
}

func (r *NotificationWsController) consumeTicket(ctx apphttp.Context, ticket string) (token string, ticketOriginHost string, ok bool) {
	cacheKey := wsTicketCachePrefix + ticket
	value := facades.Cache().GetString(cacheKey, "")
	if value == "" {
		return "", "", false
	}
	_ = facades.Cache().Forget(cacheKey)

	// Prefer 4-part (with origin host); accept legacy 3-part tickets.
	parts := strings.SplitN(value, "|", 4)
	var tenantID uint64
	switch len(parts) {
	case 4:
		var err error
		tenantID, err = strconv.ParseUint(parts[0], 10, 32)
		if err != nil {
			return "", "", false
		}
		if _, err := strconv.ParseUint(parts[1], 10, 32); err != nil {
			return "", "", false
		}
		ticketOriginHost = strings.TrimSpace(parts[2])
		token = parts[3]
	case 3:
		var err error
		tenantID, err = strconv.ParseUint(parts[0], 10, 32)
		if err != nil {
			return "", "", false
		}
		if _, err := strconv.ParseUint(parts[1], 10, 32); err != nil {
			return "", "", false
		}
		token = parts[2]
	default:
		return "", "", false
	}
	if token == "" {
		return "", "", false
	}
	if tenancy.Enabled() {
		if tenantID == 0 {
			logger.WarnfHTTP(ctx, "WebSocket ticket has no tenant_id (re-issue ticket with X-Tenant-ID)")
			return "", "", false
		}
		// Bind by id only: BindHTTP(id) + Origin code (acme.*) used to raise tenant_hint_conflict.
		if err := services.NewTenantConnectionService().BindHTTPByTenantID(ctx, uint(tenantID)); err != nil {
			logger.WarnfHTTP(ctx, "WebSocket ticket tenant bind failed: tenant_id=%d err=%v", tenantID, err)
			return "", "", false
		}
	}
	return token, ticketOriginHost, true
}

func (r *NotificationWsController) currentAdmin(ctx apphttp.Context) *models.Admin {
	if adminValue := ctx.Value("admin"); adminValue != nil {
		if admin, ok := adminValue.(models.Admin); ok {
			return &admin
		}
		if adminPtr, ok := adminValue.(*models.Admin); ok {
			return adminPtr
		}
	}
	return nil
}

func (r *NotificationWsController) isOriginAllowed(req *http.Request, tenantID uint, ticketOriginHost string, fromTicket bool) bool {
	// Valid one-time ticket already authenticated the session. Origin allowlists
	// (DOMAINS_ADMIN / TENANCY_BASE_DOMAIN / tenant_domains) miss vanity hosts on a
	// second apex or customer CDN; do not block ticket upgrades on Origin alone.
	if fromTicket {
		return true
	}

	origin := strings.TrimSpace(req.Header.Get("Origin"))
	if origin == "" {
		return false
	}

	parsed, err := url.Parse(origin)
	if err != nil || parsed.Hostname() == "" {
		return false
	}

	originHost := tenancy.NormalizeHost(parsed.Hostname())
	// Same-host via reverse proxy: Origin host matches the request Host.
	reqHost := tenancy.NormalizeHost(req.Host)
	if i := strings.Index(reqHost, ":"); i > 0 {
		reqHost = reqHost[:i]
	}
	if reqHost != "" && originHost == reqHost {
		return true
	}

	if ticketOriginHost != "" && originHost == tenancy.NormalizeHost(ticketOriginHost) {
		return true
	}

	if appmiddleware.IsCorsOriginAllowed(origin) {
		return true
	}

	allowedAdminDomains := getConfigStringSlice("domains.admin")
	if len(allowedAdminDomains) > 0 && matchDomain(originHost, allowedAdminDomains) {
		return true
	}
	base := tenancy.BaseDomain()
	if base != "" && (originHost == base || strings.HasSuffix(originHost, "."+base)) {
		return true
	}
	if services.NewTenantDomainService().IsActiveHost(originHost) {
		return true
	}

	if tenancy.Enabled() && tenantID > 0 && r.originBelongsToTenant(originHost, tenantID) {
		return true
	}

	return false
}

// originBelongsToTenant reports whether originHost maps to the given tenant id.
func (r *NotificationWsController) originBelongsToTenant(originHost string, tenantID uint) bool {
	hint := services.NewTenantDomainService().ResolveActiveCode(originHost)
	if hint == "" {
		hint = tenancy.SubdomainHint(originHost)
	}
	if hint == "" {
		return false
	}
	tenant, err := services.NewTenantConnectionService().FindTenantByIDOrCode(hint)
	return err == nil && tenant != nil && tenant.ID == tenantID
}

// clientPageHost prefers Origin, then Referer (some same-origin proxies omit Origin).
func clientPageHost(ctx apphttp.Context) string {
	if host := originHostFromHeader(ctx.Request().Header("Origin", "")); host != "" {
		return host
	}
	return originHostFromHeader(ctx.Request().Header("Referer", ""))
}

func originHostFromHeader(origin string) string {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return ""
	}
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Hostname() == "" {
		return ""
	}
	return tenancy.NormalizeHost(parsed.Hostname())
}

func matchDomain(host string, patterns []string) bool {
	for _, pattern := range patterns {
		p := strings.TrimSpace(strings.ToLower(pattern))
		if p == "" {
			continue
		}
		if p == host {
			return true
		}
		if strings.HasPrefix(p, "*.") {
			suffix := strings.TrimPrefix(p, "*.")
			if strings.HasSuffix(host, "."+suffix) {
				return true
			}
		}
	}
	return false
}

func getConfigStringSlice(key string) []string {
	value := facades.Config().Get(key)
	switch v := value.(type) {
	case []string:
		return v
	case []any:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	default:
		return []string{}
	}
}
