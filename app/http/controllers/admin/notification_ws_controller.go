package admin

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	appfacades "goravel/app/facades"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
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
	wsTicketUsedPrefix  = "ws:ticket-used:"
	wsTicketTTL         = 60 * time.Second
	wsTicketVersion     = "v1"
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

	tenantID, _ := helpers.GetTenantIDFromContext(ctx)
	if tenancy.Enabled() && tenantID == 0 {
		logger.WarnfHTTP(ctx, "WebSocket ticket refused: tenant not bound (send X-Tenant-ID when calling api.* host)")
		return response.Error(ctx, http.StatusBadRequest, "tenant_required")
	}

	originHost := clientPageHost(ctx)
	exp := time.Now().Add(wsTicketTTL).Unix()
	ticket, err := mintSignedWsTicket(tenantID, admin.ID, exp, originHost, token)
	if err != nil {
		return response.ErrorWithLog(ctx, "notification", err, map[string]any{
			"admin_id": admin.ID,
		})
	}

	logger.InfofHTTP(ctx, "WebSocket ticket issued tenant_id=%d origin_host=%s ticket_len=%d", tenantID, originHost, len(ticket))
	// #region agent log
	debugWsNDJSON("A", "notification_ws_controller.go:Ticket", "ticket_issued", map[string]any{
		"tenant_id":   tenantID,
		"origin_host": originHost,
		"ticket_len":  len(ticket),
		"ticket_pref": ticketPrefix(ticket),
	})
	// #endregion

	return response.Success(ctx, apphttp.Json{
		"ticket":     ticket,
		"expires_in": int(wsTicketTTL / time.Second),
	})
}

// ClientLog receives browser-side WS debug events (auth required).
func (r *NotificationWsController) ClientLog(ctx apphttp.Context) apphttp.Response {
	var body map[string]any
	if err := ctx.Request().Bind(&body); err != nil {
		return response.Error(ctx, http.StatusBadRequest, "invalid_argument")
	}
	hyp, _ := body["hypothesisId"].(string)
	msg, _ := body["message"].(string)
	data, _ := body["data"].(map[string]any)
	if data == nil {
		data = map[string]any{}
	}
	data["page_host"] = originHostFromHeader(ctx.Request().Header("Origin", ""))
	// #region agent log
	debugWsNDJSON(hyp, "notification_ws_controller.go:ClientLog", msg, data)
	// #endregion
	return response.Success(ctx)
}

func (r *NotificationWsController) Server(ctx apphttp.Context) apphttp.Response {
	upgrade := ctx.Request().Header("Upgrade", "")
	connection := ctx.Request().Header("Connection", "")
	origin := ctx.Request().Header("Origin", "")
	ticketQuery := str.Of(ctx.Request().Query("ticket")).Trim().String()
	logger.InfofHTTP(ctx, "WebSocket connection attempt path=%s upgrade=%s connection=%s origin=%s ticket_len=%d",
		ctx.Request().Path(), upgrade, connection, origin, len(ticketQuery))
	// #region agent log
	debugWsNDJSON("A", "notification_ws_controller.go:Server", "ws_attempt", map[string]any{
		"upgrade":     upgrade,
		"connection":  connection,
		"origin":      origin,
		"ticket_len":  len(ticketQuery),
		"ticket_pref": ticketPrefix(ticketQuery),
		"host":        ctx.Request().Host(),
	})
	// #endregion

	isUpgrade := strings.EqualFold(strings.TrimSpace(upgrade), "websocket")

	// Non-upgrade GET must not burn one-time / signed tickets.
	if ticketQuery != "" && !isUpgrade {
		logger.WarnfHTTP(ctx, "WebSocket ticket present without Upgrade header (not consumed)")
		// #region agent log
		debugWsNDJSON("A", "notification_ws_controller.go:Server", "reject_no_upgrade", map[string]any{"ticket_len": len(ticketQuery)})
		// #endregion
		return response.Error(ctx, http.StatusBadRequest, "websocket_upgrade_required")
	}

	token, fromTicket, ticketOriginHost, ticketID, ticketErr := r.extractToken(ctx)
	if ticketErr != "" {
		logger.WarnfHTTP(ctx, "WebSocket connection rejected: %s origin=%s", ticketErr, origin)
		// #region agent log
		debugWsNDJSON("B", "notification_ws_controller.go:Server", "reject_ticket", map[string]any{
			"error": ticketErr, "origin": origin, "ticket_pref": ticketPrefix(ticketQuery),
		})
		// #endregion
		return response.Error(ctx, http.StatusUnauthorized, ticketErr)
	}
	if token == "" {
		logger.WarnfHTTP(ctx, "WebSocket connection rejected: token_required")
		// #region agent log
		debugWsNDJSON("B", "notification_ws_controller.go:Server", "reject_token_required", map[string]any{"origin": origin})
		// #endregion
		return response.Error(ctx, http.StatusUnauthorized, "token_required")
	}

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
		logger.WarnfHTTP(ctx, "WebSocket connection rejected: invalid_token from_ticket=%v", fromTicket)
		// #region agent log
		debugWsNDJSON("C", "notification_ws_controller.go:Server", "reject_invalid_token", map[string]any{
			"from_ticket": fromTicket, "err": fmt.Sprintf("%v", err),
		})
		// #endregion
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
			origin, req.Host, tenantID, ticketOriginHost, fromTicket)
		// #region agent log
		debugWsNDJSON("D", "notification_ws_controller.go:Server", "reject_origin", map[string]any{
			"origin": origin, "ticket_origin": ticketOriginHost, "from_ticket": fromTicket, "tenant_id": tenantID,
		})
		// #endregion
		return response.Error(ctx, http.StatusForbidden, "origin_not_allowed")
	}

	conn, err := websocket.Accept(ctx.Response().Writer(), req, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		logger.ErrorfHTTP(ctx, "notification ws upgrade error: %v", err)
		// #region agent log
		debugWsNDJSON("E", "notification_ws_controller.go:Server", "reject_accept", map[string]any{"err": err.Error()})
		// #endregion
		return ctx.Response().String(http.StatusInternalServerError, "upgrade_failed: "+err.Error())
	}

	if ticketID != "" {
		_ = facades.Cache().Put(wsTicketUsedPrefix+ticketID, "1", wsTicketTTL)
		_ = facades.Cache().Forget(wsTicketCachePrefix + ticketID)
	}

	wsnotifications.Hub().RegisterConnection(conn, tenantID, admin.ID)
	logger.InfofHTTP(ctx, "WebSocket connected admin_id=%d tenant_id=%d origin=%s from_ticket=%v",
		admin.ID, tenantID, origin, fromTicket)
	// #region agent log
	debugWsNDJSON("E", "notification_ws_controller.go:Server", "ws_connected", map[string]any{
		"admin_id": admin.ID, "tenant_id": tenantID, "from_ticket": fromTicket,
	})
	// #endregion

	return nil
}

func (r *NotificationWsController) extractToken(ctx apphttp.Context) (token string, fromTicket bool, ticketOriginHost, ticketID, ticketErr string) {
	ticket := str.Of(ctx.Request().Query("ticket")).Trim().String()
	if ticket != "" {
		tok, originHost, id, errCode := r.loadTicket(ctx, ticket)
		if errCode != "" {
			return "", false, "", "", errCode
		}
		return tok, true, originHost, id, ""
	}

	authorization := str.Of(ctx.Request().Header("Authorization", "")).Trim().String()
	if authorization != "" {
		return authorization, false, "", "", ""
	}

	return "", false, "", "", ""
}

func (r *NotificationWsController) loadTicket(ctx apphttp.Context, ticket string) (token, ticketOriginHost, ticketID, errCode string) {
	// Preferred: HMAC-signed ticket (works across API nodes without shared memory cache).
	if strings.HasPrefix(ticket, wsTicketVersion+".") {
		tenantID, _, exp, originHost, nonce, tok, parseErr := splitSignedPayload(ticket)
		if parseErr != "" {
			logger.WarnfHTTP(ctx, "WebSocket signed ticket rejected: %s", parseErr)
			return "", "", "", parseErr
		}
		if time.Now().Unix() > exp {
			return "", "", "", "ticket_invalid"
		}
		if facades.Cache().GetString(wsTicketUsedPrefix+nonce, "") != "" {
			logger.WarnfHTTP(ctx, "WebSocket signed ticket already used id=%s", nonce)
			return "", "", "", "ticket_invalid"
		}
		if tenancy.Enabled() {
			if tenantID == 0 {
				return "", "", "", "tenant_required"
			}
			if err := services.NewTenantConnectionService().BindHTTPByTenantID(ctx, tenantID); err != nil {
				logger.WarnfHTTP(ctx, "WebSocket ticket tenant bind failed: tenant_id=%d err=%v", tenantID, err)
				if businessErr, ok := apperrors.GetBusinessError(err); ok {
					return "", "", "", businessErr.Code
				}
				return "", "", "", "tenant_required"
			}
		}
		return tok, originHost, nonce, ""
	}

	// Legacy ULID cache tickets (single-node memory / older builds).
	ticketID = strings.ToLower(ticket)
	cacheKey := wsTicketCachePrefix + ticketID
	value := facades.Cache().GetString(cacheKey, "")
	if value == "" {
		logger.WarnfHTTP(ctx, "WebSocket ticket missing or expired ticket=%s", ticketID)
		return "", "", "", "ticket_invalid"
	}

	parts := strings.SplitN(value, "|", 4)
	var tenantID uint64
	switch len(parts) {
	case 4:
		var err error
		tenantID, err = strconv.ParseUint(parts[0], 10, 32)
		if err != nil {
			return "", "", "", "ticket_invalid"
		}
		if _, err := strconv.ParseUint(parts[1], 10, 32); err != nil {
			return "", "", "", "ticket_invalid"
		}
		ticketOriginHost = strings.TrimSpace(parts[2])
		token = parts[3]
	case 3:
		var err error
		tenantID, err = strconv.ParseUint(parts[0], 10, 32)
		if err != nil {
			return "", "", "", "ticket_invalid"
		}
		if _, err := strconv.ParseUint(parts[1], 10, 32); err != nil {
			return "", "", "", "ticket_invalid"
		}
		token = parts[2]
	default:
		return "", "", "", "ticket_invalid"
	}
	if token == "" {
		return "", "", "", "ticket_invalid"
	}
	if tenancy.Enabled() {
		if tenantID == 0 {
			return "", "", "", "tenant_required"
		}
		if err := services.NewTenantConnectionService().BindHTTPByTenantID(ctx, uint(tenantID)); err != nil {
			logger.WarnfHTTP(ctx, "WebSocket ticket tenant bind failed: tenant_id=%d err=%v", tenantID, err)
			if businessErr, ok := apperrors.GetBusinessError(err); ok {
				return "", "", "", businessErr.Code
			}
			return "", "", "", "tenant_required"
		}
	}
	return token, ticketOriginHost, ticketID, ""
}

func mintSignedWsTicket(tenantID, adminID uint, exp int64, originHost, token string) (string, error) {
	key := strings.TrimSpace(facades.Config().GetString("app.key"))
	if key == "" {
		return "", fmt.Errorf("APP_KEY is empty")
	}
	nonce := strings.ToLower(ulid.Make().String())
	// version|tenant|admin|exp|origin|nonce|token
	payload := fmt.Sprintf("%s|%d|%d|%d|%s|%s|%s",
		wsTicketVersion, tenantID, adminID, exp, originHost, nonce, token)
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write([]byte(payload))
	sig := hex.EncodeToString(mac.Sum(nil))
	return wsTicketVersion + "." + base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + sig, nil
}

func splitSignedPayload(ticket string) (tenantID uint, adminID uint, exp int64, originHost, nonce, token, errCode string) {
	parts := strings.Split(ticket, ".")
	if len(parts) != 3 || parts[0] != wsTicketVersion {
		return 0, 0, 0, "", "", "", "ticket_invalid"
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return 0, 0, 0, "", "", "", "ticket_invalid"
	}
	key := strings.TrimSpace(facades.Config().GetString("app.key"))
	if key == "" {
		return 0, 0, 0, "", "", "", "ticket_invalid"
	}
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write(raw)
	want := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(want), []byte(parts[2])) {
		return 0, 0, 0, "", "", "", "ticket_invalid"
	}
	fields := strings.SplitN(string(raw), "|", 7)
	if len(fields) != 7 || fields[0] != wsTicketVersion {
		return 0, 0, 0, "", "", "", "ticket_invalid"
	}
	tid, err := strconv.ParseUint(fields[1], 10, 32)
	if err != nil {
		return 0, 0, 0, "", "", "", "ticket_invalid"
	}
	aid, err := strconv.ParseUint(fields[2], 10, 32)
	if err != nil {
		return 0, 0, 0, "", "", "", "ticket_invalid"
	}
	expVal, err := strconv.ParseInt(fields[3], 10, 64)
	if err != nil {
		return 0, 0, 0, "", "", "", "ticket_invalid"
	}
	return uint(tid), uint(aid), expVal, fields[4], fields[5], fields[6], ""
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

func ticketPrefix(ticket string) string {
	if len(ticket) <= 24 {
		return ticket
	}
	return ticket[:12] + "..." + ticket[len(ticket)-8:]
}

// #region agent log
func debugWsNDJSON(hypothesisId, location, message string, data map[string]any) {
	payload := map[string]any{
		"sessionId":    "79bec9",
		"hypothesisId": hypothesisId,
		"location":     location,
		"message":      message,
		"data":         data,
		"timestamp":    time.Now().UnixMilli(),
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return
	}
	line := string(b)
	// Always mirror into application log so operators can grep without fighting cwd.
	facades.Log().Infof("DBG79bec9 %s", line)

	paths := []string{
		facades.App().StoragePath("logs", "debug-79bec9.log"),
		filepath.Join("storage", "logs", "debug-79bec9.log"),
		"/tmp/debug-79bec9.log",
	}
	raw := append([]byte(line), '\n')
	for _, p := range paths {
		if p == "" {
			continue
		}
		_ = os.MkdirAll(filepath.Dir(p), 0755)
		f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			continue
		}
		_, _ = f.Write(raw)
		_ = f.Close()
	}
}

// #endregion
