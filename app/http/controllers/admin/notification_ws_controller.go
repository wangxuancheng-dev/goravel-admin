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

type NotificationWsController struct {}


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
	cacheKey := wsTicketCachePrefix + ticket
	cacheValue := fmt.Sprintf("%d|%d|%s", tenantID, admin.ID, token)
	if err := facades.Cache().Put(cacheKey, cacheValue, wsTicketTTL); err != nil {
		return response.ErrorWithLog(ctx, "notification", err, map[string]any{
			"admin_id": admin.ID,
		})
	}

	return response.Success(ctx, apphttp.Json{
		"ticket":     ticket,
		"expires_in": int(wsTicketTTL / time.Second),
	})
}

func (r *NotificationWsController) Server(ctx apphttp.Context) apphttp.Response {
	// 记录 WebSocket 连接尝试（仅 Debug 模式）
	logger.DebugfHTTP(ctx, "WebSocket connection attempt from %s, path: %s, upgrade: %s, connection: %s",
		ctx.Request().Ip(),
		ctx.Request().Path(),
		ctx.Request().Header("Upgrade", ""),
		ctx.Request().Header("Connection", ""))

	token := r.extractToken(ctx)
	if token == "" {
		logger.WarnfHTTP(ctx, "WebSocket connection rejected: token required")
		response.Error(ctx, http.StatusUnauthorized, "token_required")
		ctx.Request().Abort()
		return nil
	}

	// Non-ticket Authorization path: bind tenant from header/query if still unbound.
	if tenancy.Enabled() {
		if _, ok := helpers.GetTenantIDFromContext(ctx); !ok {
			if err := services.NewTenantConnectionService().BindHTTP(ctx, ""); err != nil {
				logger.WarnfHTTP(ctx, "WebSocket tenant bind failed: %v", err)
				if businessErr, ok := apperrors.GetBusinessError(err); ok {
					response.Error(ctx, http.StatusBadRequest, businessErr.Code)
				} else {
					response.Error(ctx, http.StatusBadRequest, "tenant_required")
				}
				ctx.Request().Abort()
				return nil
			}
		}
	}

	token = str.Of(token).ChopStart("Bearer ").Trim().String()
	accessToken, err := r.tokenService(ctx).FindToken(token)
	if err != nil || accessToken == nil || accessToken.TokenableType != "admin" {
		response.Error(ctx, http.StatusUnauthorized, "invalid_token")
		ctx.Request().Abort()
		return nil
	}

	var admin models.Admin
	if err := appfacades.OrmQuery(ctx).Where("id", accessToken.TokenableID).FirstOrFail(&admin); err != nil {
		response.Error(ctx, http.StatusUnauthorized, "user_not_found")
		ctx.Request().Abort()
		return nil
	}
	_ = r.tokenService(ctx).UpdateLastUsedAt(token)

	req := ctx.Request().Origin()
	if !r.isOriginAllowed(req) {
		logger.WarnfHTTP(ctx, "WebSocket connection rejected: origin not allowed")
		response.Error(ctx, http.StatusForbidden, "origin_not_allowed")
		ctx.Request().Abort()
		return nil
	}

	// Origin already verified above; skip library check.
	conn, err := websocket.Accept(ctx.Response().Writer(), req, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		logger.ErrorfHTTP(ctx, "notification ws upgrade error: %v", err)
		return ctx.Response().String(http.StatusInternalServerError, "upgrade_failed")
	}

	tenantID, _ := helpers.GetTenantIDFromContext(ctx)
	wsnotifications.Hub().RegisterConnection(conn, tenantID, admin.ID)

	return nil
}

func (r *NotificationWsController) extractToken(ctx apphttp.Context) string {
	ticket := str.Of(ctx.Request().Query("ticket")).Trim().String()
	if ticket != "" {
		token, ok := r.consumeTicket(ctx, ticket)
		if ok {
			return token
		}
		logger.WarnfHTTP(ctx, "WebSocket ticket invalid or expired")
	}

	// Keep header parsing only for non-browser clients.
	authorization := str.Of(ctx.Request().Header("Authorization", "")).Trim().String()
	if authorization != "" {
		return authorization
	}

	return ""
}

func (r *NotificationWsController) consumeTicket(ctx apphttp.Context, ticket string) (string, bool) {
	cacheKey := wsTicketCachePrefix + ticket
	value := facades.Cache().GetString(cacheKey, "")
	if value == "" {
		return "", false
	}
	_ = facades.Cache().Forget(cacheKey)

	parts := strings.SplitN(value, "|", 3)
	if len(parts) != 3 {
		return "", false
	}
	tenantID, err := strconv.ParseUint(parts[0], 10, 32)
	if err != nil {
		return "", false
	}
	if _, err := strconv.ParseUint(parts[1], 10, 32); err != nil {
		return "", false
	}
	token := parts[2]
	if token == "" {
		return "", false
	}
	if tenancy.Enabled() && tenantID > 0 {
		if err := services.NewTenantConnectionService().BindHTTP(ctx, fmt.Sprintf("%d", tenantID)); err != nil {
			logger.WarnfHTTP(ctx, "WebSocket ticket tenant bind failed: tenant_id=%d err=%v", tenantID, err)
			return "", false
		}
	}
	return token, true
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

func (r *NotificationWsController) isOriginAllowed(req *http.Request) bool {
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

	allowedAdminDomains := getConfigStringSlice("domains.admin")
	if len(allowedAdminDomains) > 0 && !matchDomain(originHost, allowedAdminDomains) {
		base := tenancy.BaseDomain()
		underBase := base != "" && (originHost == base || strings.HasSuffix(originHost, "."+base))
		if !underBase && !services.NewTenantDomainService().IsActiveHost(originHost) {
			return false
		}
	}

	return appmiddleware.IsCorsOriginAllowed(origin)
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
