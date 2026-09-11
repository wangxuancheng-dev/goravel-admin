package middleware

import (
	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"

	"goravel/app/http/response"
	"goravel/app/models"
)

// UserJwt C端用户JWT认证中间件（使用Goravel标准Auth）
func UserJwt() http.Middleware {
	return newMiddleware("user_jwt", func(ctx http.Context) {
		// 使用Goravel标准Auth解析token
		if _, err := facades.Auth(ctx).Guard("user").Parse(ctx.Request().Header("Authorization", "")); err != nil {
			// 如果Header中没有token，尝试从URL参数中获取
			if token := ctx.Request().Query("_token", ""); token != "" {
				if _, err := facades.Auth(ctx).Guard("user").Parse(token); err != nil {
					response.Abort(ctx, http.StatusUnauthorized, "invalid_token")
					return
				}
			} else {
				response.Abort(ctx, http.StatusUnauthorized, "not_logged_in")
				return
			}
		}

		// 获取用户信息
		var user models.User
		if err := facades.Auth(ctx).Guard("user").User(&user); err != nil {
			response.Abort(ctx, http.StatusUnauthorized, "user_not_found")
			return
		}

		if user.ID == 0 {
			response.Abort(ctx, http.StatusUnauthorized, "user_not_found")
			return
		}

		// 检查用户状态
		if user.Status == 0 {
			response.Abort(ctx, http.StatusForbidden, "account_disabled")
			return
		}

		// 将用户信息存储到context中，供后续中间件使用
		ctx.WithValue("user", user)

		ctx.Request().Next()
	})
}
