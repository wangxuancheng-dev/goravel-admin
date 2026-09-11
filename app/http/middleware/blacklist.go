package middleware

import (
	appfacades "goravel/app/facades"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"

	"goravel/app/http/helpers"
	"goravel/app/http/response"
	"goravel/app/models"
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

		// 获取真实IP地址
		realIP := helpers.GetRealIP(ctx)

		// 查询所有启用的黑名单记录
		var blacklists []models.Blacklist
		if err := appfacades.OrmQuery(ctx).Where("status", 1).Get(&blacklists); err != nil {
			// 如果查询失败，记录错误但继续处理请求（避免影响系统正常运行）
			facades.Log().Errorf("Blacklist middleware: Failed to query blacklists: %v", err)
			ctx.Request().Next()
			return
		}

		// 检查IP是否在黑名单中
		for _, blacklist := range blacklists {
			if utils.IsIPInBlacklist(realIP, blacklist.IP) {
				// IP在黑名单中，拒绝访问
				facades.Log().Warningf("Blacklist middleware: IP %s blocked by blacklist ID %d", realIP, blacklist.ID)
				response.Abort(ctx, http.StatusForbidden, "ip_blocked")
				return
			}
		}

		// IP不在黑名单中，继续处理请求
		ctx.Request().Next()
	})
}
