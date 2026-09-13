package providers

import (
	"fmt"
	"runtime/debug"
	"strconv"
	"strings"
	"testing"

	"github.com/goravel/framework/contracts/foundation"
	contractshttp "github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/http/limit"

	"goravel/app/facades"
	"goravel/app/http"
	"goravel/app/http/helpers"
	"goravel/app/http/response"
	"goravel/app/http/trans"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/app/tenancy"
	"goravel/app/tenancyctx"
)

type RouteServiceProvider struct {
}

func (receiver *RouteServiceProvider) Register(app foundation.Application) {
}

func (receiver *RouteServiceProvider) Boot(app foundation.Application) {
	// Add HTTP middleware
	facades.Route().GlobalMiddleware(http.Kernel{}.Middleware()...)
	facades.Route().Recover(func(ctx contractshttp.Context, err any) {
		// Nested recover: logging / Abort must never escape as http.Server panic.
		// gin-contrib/timeout re-throws panics; a secondary nil deref here used to
		// surface as "http: panic serving".
		defer func() {
			if rec := recover(); rec != nil {
				facades.Log().Errorf("recover callback panicked: %v (original: %v)\n%s", rec, err, debug.Stack())
				safeAbort(ctx, contractshttp.StatusInternalServerError, "operation_failed")
			}
		}()

		msg := fmt.Sprintf("%v", err)
		// Malformed client bodies / scanners (goravel/gin getHttpBody): do not flood system_logs.
		if isBadRequestBodyPanic(err) {
			facades.Log().Warning(msg)
			safeAbort(ctx, contractshttp.StatusBadRequest, "params_error")
			return
		}

		systemLogService := services.NewSystemLogService(ctx)
		_ = systemLogService.RecordHTTP(ctx, "error", "recover", msg, map[string]any{
			"stack": string(debug.Stack()),
		})
		facades.Log().Errorf("request panic recovered: %v\n%s", err, debug.Stack())
		safeAbort(ctx, contractshttp.StatusInternalServerError, "operation_failed")
	})

	receiver.configureRateLimiting()
}

// safeAbort calls response.Abort and swallows panics (e.g. broken Writer after timeout).
func safeAbort(ctx contractshttp.Context, code int, messageOrErr any) {
	defer func() { _ = recover() }()
	if ctx == nil {
		return
	}
	response.Abort(ctx, code, messageOrErr)
}

// isBadRequestBodyPanic reports recover payloads that typically come from
// malformed client request bodies (scanners / truncated multipart).
// Generic nil-pointer panics are only treated as client errors when the stack
// points at goravel/gin getHttpBody (upstream MultipartForm nil bug).
func isBadRequestBodyPanic(err any) bool {
	return isBadRequestBodyPanicWithStack(fmt.Sprintf("%v", err), string(debug.Stack()))
}

func isBadRequestBodyPanicWithStack(msg, stack string) bool {
	msg = strings.ToLower(msg)
	if strings.Contains(msg, "parse multipart form error") ||
		strings.Contains(msg, "malformed mime header") ||
		strings.Contains(msg, "multipart: nextpart") ||
		strings.Contains(msg, "request body too large") {
		return true
	}
	if strings.Contains(msg, "nil pointer") || strings.Contains(msg, "invalid memory address") {
		stack = strings.ToLower(stack)
		return strings.Contains(stack, "gethttpbody") ||
			strings.Contains(stack, "parsemultipartform")
	}
	return false
}

func (receiver *RouteServiceProvider) configureRateLimiting() {
	// 全局速率限制器
	facades.RateLimiter().For("global", func(ctx contractshttp.Context) contractshttp.Limit {
		return limit.PerMinute(1000)
	})

	// IP 速率限制器
	facades.RateLimiter().ForWithLimits("ip", func(ctx contractshttp.Context) []contractshttp.Limit {
		return []contractshttp.Limit{
			limit.PerDay(1000),
			limit.PerMinute(2).By(ctx.Request().Ip()),
		}
	})

	// Login rate limiter: IP + scope (platform/admin/user) + tenant hint + account.
	// Platform and tenant admins often share usernames; scopes must not share buckets.
	facades.RateLimiter().For("login", func(ctx contractshttp.Context) contractshttp.Limit {
		ip := helpers.GetRealIP(ctx)
		username := resolveLoginIdentifier(ctx, ip)
		scope := resolveLoginScope(ctx)
		tenantHint := resolveLoginTenantHint(ctx)
		perMinute := 6
		// Feature / unit HTTP tests issue many logins from one IP.
		if facades.Config().GetString("app.env") == "test" || testing.Testing() {
			perMinute = 1000
		}

		key := buildLoginRateLimitKey(ip, scope, tenantHint, username)
		return limit.PerMinute(perMinute).Response(func(ctx contractshttp.Context) {
			response.Abort(ctx, contractshttp.StatusTooManyRequests, "too_many_requests")
		}).By(key)
	})

	// 测试响应速率限制器（仅开发环境使用）
	facades.RateLimiter().For("testResponse", func(ctx contractshttp.Context) contractshttp.Limit {
		return limit.PerMinute(6).Response(func(ctx contractshttp.Context) {
			response.Abort(ctx, contractshttp.StatusTooManyRequests, "too_many_requests")
		})
	})

	// pprof token verify (per tenant + admin + IP)
	facades.RateLimiter().For("pprofVerify", func(ctx contractshttp.Context) contractshttp.Limit {
		ip := helpers.GetRealIP(ctx)
		identifier := resolvePprofVerifyIdentifier(ctx, ip)
		return limit.PerMinute(6).Response(func(ctx contractshttp.Context) {
			_ = ctx.Response().Json(contractshttp.StatusTooManyRequests, contractshttp.Json{
				"code":       contractshttp.StatusTooManyRequests,
				"message":    trans.Get(ctx, "too_many_requests"),
				"error_code": "pprof_verify_rate_limited",
			}).Abort()
		}).By(tenantRateKeyPrefix(ctx) + ip + ":pprof_verify:" + identifier)
	})

	// pprof CPU sample (per tenant + admin + IP)
	facades.RateLimiter().For("pprofCPU", func(ctx contractshttp.Context) contractshttp.Limit {
		ip := helpers.GetRealIP(ctx)
		identifier := resolvePprofVerifyIdentifier(ctx, ip)
		return limit.PerMinute(3).Response(func(ctx contractshttp.Context) {
			_ = ctx.Response().Json(contractshttp.StatusTooManyRequests, contractshttp.Json{
				"code":       contractshttp.StatusTooManyRequests,
				"message":    trans.Get(ctx, "too_many_requests"),
				"error_code": "pprof_cpu_rate_limited",
			}).Abort()
		}).By(tenantRateKeyPrefix(ctx) + ip + ":pprof_cpu:" + identifier)
	})

	// pprof memory sample (per tenant + admin + IP)
	facades.RateLimiter().For("pprofMemory", func(ctx contractshttp.Context) contractshttp.Limit {
		ip := helpers.GetRealIP(ctx)
		identifier := resolvePprofVerifyIdentifier(ctx, ip)
		return limit.PerMinute(6).Response(func(ctx contractshttp.Context) {
			_ = ctx.Response().Json(contractshttp.StatusTooManyRequests, contractshttp.Json{
				"code":       contractshttp.StatusTooManyRequests,
				"message":    trans.Get(ctx, "too_many_requests"),
				"error_code": "pprof_memory_rate_limited",
			}).Abort()
		}).By(tenantRateKeyPrefix(ctx) + ip + ":pprof_memory:" + identifier)
	})

	// AI 实验室限流（按管理员账号：分钟 + 日配额）
	facades.RateLimiter().ForWithLimits("aiLab", func(ctx contractshttp.Context) []contractshttp.Limit {
		adminID := resolveAdminIdentifier(ctx)
		perMinute := facades.Config().GetInt("ai.lab_rate_limit_per_minute", 10)
		perDay := facades.Config().GetInt("ai.lab_rate_limit_per_day", 200)
		if perMinute < 1 {
			perMinute = 1
		}
		if perDay < 1 {
			perDay = 1
		}

		rateLimited := func(ctx contractshttp.Context) {
			response.Abort(ctx, contractshttp.StatusTooManyRequests, "ai_lab_rate_limited")
		}

		key := tenantRateKeyPrefix(ctx) + "ai_lab:admin:" + adminID
		return []contractshttp.Limit{
			limit.PerMinute(perMinute).Response(rateLimited).By(key + ":minute"),
			limit.PerDay(perDay).Response(rateLimited).By(key + ":day"),
		}
	})

	// 后台已认证 API 角色限流
	facades.RateLimiter().For("adminApi", func(ctx contractshttp.Context) contractshttp.Limit {
		perMinute := resolveAdminAPIRateLimit(ctx)
		adminID := resolveAdminIdentifier(ctx)
		return limit.PerMinute(perMinute).Response(func(ctx contractshttp.Context) {
			response.Abort(ctx, contractshttp.StatusTooManyRequests, "too_many_requests")
		}).By(tenantRateKeyPrefix(ctx) + "admin_api:" + adminID)
	})
}

// resolveAdminAPIRateLimit 按角色与请求方法选择限流阈值。
func resolveAdminAPIRateLimit(ctx contractshttp.Context) int {
	cfg := facades.Config()
	superLimit := cfg.GetInt("login_security.role_rate_limits.super_admin_per_minute", 1200)
	defaultLimit := cfg.GetInt("login_security.role_rate_limits.default_per_minute", 300)
	writeLimit := cfg.GetInt("login_security.role_rate_limits.write_per_minute", 120)
	if superLimit < 1 {
		superLimit = 1200
	}
	if defaultLimit < 1 {
		defaultLimit = 300
	}
	if writeLimit < 1 {
		writeLimit = 120
	}

	if isSuperAdminFromContext(ctx) {
		return superLimit
	}

	method := strings.ToUpper(ctx.Request().Method())
	switch method {
	case "POST", "PUT", "PATCH", "DELETE":
		return writeLimit
	default:
		return defaultLimit
	}
}

func isSuperAdminFromContext(ctx contractshttp.Context) bool {
	adminValue := ctx.Value("admin")
	if adminValue == nil {
		return false
	}

	var admin models.Admin
	switch v := adminValue.(type) {
	case models.Admin:
		admin = v
	case *models.Admin:
		if v == nil {
			return false
		}
		admin = *v
	default:
		return false
	}

	if len(admin.Roles) == 0 {
		_ = facades.OrmQuery(ctx).Where("id", admin.ID).With("Roles").First(&admin)
	}
	for _, role := range admin.Roles {
		if role.Slug == "super-admin" && role.Status == 1 {
			return true
		}
	}
	return false
}

// tenantRateKeyPrefix isolates rate-limit buckets across tenants (admin IDs collide).
func tenantRateKeyPrefix(ctx contractshttp.Context) string {
	if !tenancy.Enabled() {
		return ""
	}
	if code, ok := tenancyctx.CodeFrom(ctx); ok && code != "" {
		return "t:" + code + ":"
	}
	if id, ok := tenancyctx.IDFrom(ctx); ok && id > 0 {
		return fmt.Sprintf("t%d:", id)
	}
	return "t_unbound:"
}

// resolveLoginIdentifier 从请求中提取登录标识（username > email > X-Username > IP fallback）。
func resolveLoginIdentifier(ctx contractshttp.Context, fallbackIP string) string {
	for _, field := range []string{"username", "email"} {
		if v := strings.TrimSpace(ctx.Request().Input(field, "")); v != "" {
			return strings.ToLower(v)
		}
	}
	if v := strings.TrimSpace(ctx.Request().Header("X-Username", "")); v != "" {
		return strings.ToLower(v)
	}
	return fallbackIP
}

// resolveLoginScope isolates platform / tenant-admin / C-end user login buckets by path.
func resolveLoginScope(ctx contractshttp.Context) string {
	return loginScopeFromPath(ctx.Request().Path())
}

func loginScopeFromPath(path string) string {
	path = strings.ToLower(strings.TrimSpace(path))
	switch {
	case strings.HasPrefix(path, "/api/platform"):
		return "platform"
	case strings.HasPrefix(path, "/api/admin"):
		return "admin"
	case strings.HasPrefix(path, "/api/user"):
		return "user"
	default:
		return "other"
	}
}

// buildLoginRateLimitKey builds the throttle bucket key for login endpoints.
func buildLoginRateLimitKey(ip, scope, tenantHint, username string) string {
	ip = strings.TrimSpace(ip)
	scope = strings.TrimSpace(scope)
	if scope == "" {
		scope = "other"
	}
	username = strings.TrimSpace(username)
	tenantHint = strings.ToLower(strings.TrimSpace(tenantHint))
	if tenantHint != "" {
		return ip + ":login:" + scope + ":" + tenantHint + ":" + username
	}
	return ip + ":login:" + scope + ":" + username
}

// resolveLoginTenantHint includes tenant in the rate-limit key (subdomain / ResolveHint first).
// When ResolveHint strips client hints on apex hosts, fall back to body/header so tenants still isolate.
func resolveLoginTenantHint(ctx contractshttp.Context) string {
	body := ""
	for _, field := range []string{"tenant_code", "tenant_id"} {
		if v := strings.TrimSpace(ctx.Request().Input(field, "")); v != "" {
			body = v
			break
		}
	}
	hint, _ := tenancy.ResolveHint(ctx, body)
	if hint = strings.ToLower(strings.TrimSpace(hint)); hint != "" {
		return hint
	}
	if body != "" {
		return strings.ToLower(body)
	}
	if h := strings.ToLower(strings.TrimSpace(tenancy.ClientHint(ctx))); h != "" {
		return h
	}
	return ""
}

// resolvePprofVerifyIdentifier 从上下文提取管理员 ID，找不到则回退到 IP
func resolvePprofVerifyIdentifier(ctx contractshttp.Context, fallbackIP string) string {
	return resolveAdminIdentifierWithFallback(ctx, fallbackIP)
}

// resolveAdminIdentifier 从上下文提取管理员 ID，找不到则回退到 IP（用于 AI 实验室限流等）
func resolveAdminIdentifier(ctx contractshttp.Context) string {
	return resolveAdminIdentifierWithFallback(ctx, helpers.GetRealIP(ctx))
}

func resolveAdminIdentifierWithFallback(ctx contractshttp.Context, fallbackIP string) string {
	adminValue := ctx.Value("admin")
	if adminValue == nil {
		return fallbackIP
	}

	if admin, ok := adminValue.(models.Admin); ok {
		return strconv.FormatUint(uint64(admin.ID), 10)
	}
	if adminPtr, ok := adminValue.(*models.Admin); ok && adminPtr != nil {
		return strconv.FormatUint(uint64(adminPtr.ID), 10)
	}

	return fallbackIP
}
