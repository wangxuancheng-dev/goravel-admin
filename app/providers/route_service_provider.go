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
)

type RouteServiceProvider struct {
}

func (receiver *RouteServiceProvider) Register(app foundation.Application) {
}

func (receiver *RouteServiceProvider) Boot(app foundation.Application) {
	// Add HTTP middleware
	facades.Route().GlobalMiddleware(http.Kernel{}.Middleware()...)
	facades.Route().Recover(func(ctx contractshttp.Context, err any) {
		msg := fmt.Sprintf("%v", err)
		// Malformed client bodies / scanners (goravel/gin getHttpBody): do not flood system_logs.
		if isBadRequestBodyPanic(err) {
			facades.Log().Warning(msg)
			response.Abort(ctx, contractshttp.StatusBadRequest, "params_error")
			return
		}

		systemLogService := services.NewSystemLogService(ctx)
		_ = systemLogService.RecordHTTP(ctx, "error", "recover", msg, nil)
		facades.Log().Error(err)
		response.Abort(ctx, contractshttp.StatusInternalServerError, "operation_failed")
	})

	receiver.configureRateLimiting()
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

	// 登录速率限制器（IP + 账号 双维度，避免攻击者锁住其他 IP 的同名账号）
	facades.RateLimiter().For("login", func(ctx contractshttp.Context) contractshttp.Limit {
		ip := helpers.GetRealIP(ctx)
		username := resolveLoginIdentifier(ctx, ip)
		perMinute := 6
		// Feature / unit HTTP tests issue many logins from one IP.
		if facades.Config().GetString("app.env") == "test" || testing.Testing() {
			perMinute = 1000
		}

		return limit.PerMinute(perMinute).Response(func(ctx contractshttp.Context) {
			response.Abort(ctx, contractshttp.StatusTooManyRequests, "too_many_requests")
		}).By(ip + ":login:" + username)
	})

	// 测试响应速率限制器（仅开发环境使用）
	facades.RateLimiter().For("testResponse", func(ctx contractshttp.Context) contractshttp.Limit {
		return limit.PerMinute(6).Response(func(ctx contractshttp.Context) {
			response.Abort(ctx, contractshttp.StatusTooManyRequests, "too_many_requests")
		})
	})

	// pprof token 验证限流（按管理员 + IP）
	facades.RateLimiter().For("pprofVerify", func(ctx contractshttp.Context) contractshttp.Limit {
		ip := helpers.GetRealIP(ctx)
		identifier := resolvePprofVerifyIdentifier(ctx, ip)
		return limit.PerMinute(6).Response(func(ctx contractshttp.Context) {
			_ = ctx.Response().Json(contractshttp.StatusTooManyRequests, contractshttp.Json{
				"code":       contractshttp.StatusTooManyRequests,
				"message":    trans.Get(ctx, "too_many_requests"),
				"error_code": "pprof_verify_rate_limited",
			}).Abort()
		}).By(ip + ":pprof_verify:" + identifier)
	})

	// pprof CPU 采样限流（按管理员 + IP）
	facades.RateLimiter().For("pprofCPU", func(ctx contractshttp.Context) contractshttp.Limit {
		ip := helpers.GetRealIP(ctx)
		identifier := resolvePprofVerifyIdentifier(ctx, ip)
		return limit.PerMinute(3).Response(func(ctx contractshttp.Context) {
			_ = ctx.Response().Json(contractshttp.StatusTooManyRequests, contractshttp.Json{
				"code":       contractshttp.StatusTooManyRequests,
				"message":    trans.Get(ctx, "too_many_requests"),
				"error_code": "pprof_cpu_rate_limited",
			}).Abort()
		}).By(ip + ":pprof_cpu:" + identifier)
	})

	// pprof 内存采样限流（按管理员 + IP）
	facades.RateLimiter().For("pprofMemory", func(ctx contractshttp.Context) contractshttp.Limit {
		ip := helpers.GetRealIP(ctx)
		identifier := resolvePprofVerifyIdentifier(ctx, ip)
		return limit.PerMinute(6).Response(func(ctx contractshttp.Context) {
			_ = ctx.Response().Json(contractshttp.StatusTooManyRequests, contractshttp.Json{
				"code":       contractshttp.StatusTooManyRequests,
				"message":    trans.Get(ctx, "too_many_requests"),
				"error_code": "pprof_memory_rate_limited",
			}).Abort()
		}).By(ip + ":pprof_memory:" + identifier)
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

		key := "ai_lab:admin:" + adminID
		return []contractshttp.Limit{
			limit.PerMinute(perMinute).Response(rateLimited).By(key + ":minute"),
			limit.PerDay(perDay).Response(rateLimited).By(key + ":day"),
		}
	})
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
