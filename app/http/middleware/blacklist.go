package middleware

import (
	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"

	"goravel/app/http/helpers"
	"goravel/app/http/response"
	"goravel/app/services"
	"goravel/app/tenancy"
	"goravel/app/utils"
)

func Blacklist() http.Middleware {
	return newMiddleware("blacklist", func(ctx http.Context) {
		// 排除登录接口，避免管理员被封禁后无法登录
		path := ctx.Request().Path()
		if path == "/api/admin/login" || path == "/api/admin/login/captcha" {
			ctx.Request().Next()
			return
		}

		// tenancy 开启且尚未绑定租户时跳过（全局中间件早于 Tenant）；路由组内 Tenant 之后会再跑一遍。
		if tenancy.Enabled() && !helpers.TenantBound(ctx) {
			ctx.Request().Next()
			return
		}

		realIP := helpers.GetRealIP(ctx)

		patterns, err := services.EnabledBlacklistPatterns(ctx)
		if err != nil {
			// 无可用缓存时 fail-closed，避免封禁名单失效被绕过
			facades.Log().Errorf("Blacklist middleware: Failed to load blacklists: %v", err)
			response.Abort(ctx, http.StatusServiceUnavailable, "service_unavailable")
			return
		}

		for _, pattern := range patterns {
			if utils.IsIPInBlacklist(realIP, pattern) {
				facades.Log().Warningf("Blacklist middleware: IP %s blocked by pattern %q", realIP, pattern)
				response.Abort(ctx, http.StatusForbidden, "ip_blocked")
				return
			}
		}

		ctx.Request().Next()
	})
}
