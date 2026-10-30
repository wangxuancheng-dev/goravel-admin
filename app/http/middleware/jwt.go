package middleware

import (
	appfacades "goravel/app/facades"
	"strings"
	"time"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"
	"github.com/goravel/framework/support/str"

	apperrors "goravel/app/errors"
	"goravel/app/http/response"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/app/utils/logger"
)

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func Jwt() http.Middleware {
	return newMiddleware("jwt", func(ctx http.Context) {
		token := ctx.Request().Header("Authorization", "")

		// 如果 Header 中没有 token，尝试从 URL 参数中获取（用于 SSE 等不支持自定义 headers 的场景）
		if str.Of(token).IsEmpty() {
			token = ctx.Request().Query("_token", "")
		}

		if str.Of(token).IsEmpty() {
			response.Abort(ctx, http.StatusUnauthorized, "not_logged_in")
			return
		}

		// 移除Bearer前缀（如果有）
		token = str.Of(token).ChopStart("Bearer ").Trim().String()

		if token == "" {
			response.Abort(ctx, http.StatusUnauthorized, "not_logged_in")
			return
		}

		// 从数据库查找token
		tokenService := services.NewTokenServiceImpl(ctx)
		accessToken, err := tokenService.FindToken(token)
		if err != nil {
			tokenPrefix := token[:min(20, len(token))]
			if !isExpectedUnauthorizedErr(err) {
				// The lookup itself failed (DB down/overloaded): a 401 would log the user out
				// and look like a client error to monitoring.
				logger.ErrorfHTTP(ctx, "JWT middleware: FindToken error: %v, token prefix: %s", err, tokenPrefix)
				response.Abort(ctx, http.StatusServiceUnavailable, "service_unavailable")
				return
			}
			logger.WarnfHTTP(ctx, "JWT middleware: unauthorized token: %v, token prefix: %s", err, tokenPrefix)
			response.Abort(ctx, http.StatusUnauthorized, "invalid_token")
			return
		}
		if accessToken == nil {
			logger.WarnfHTTP(ctx, "JWT middleware: accessToken is nil, token prefix: %s", token[:min(20, len(token))])
			response.Abort(ctx, http.StatusUnauthorized, "invalid_token")
			return
		}

		// 检查token类型
		if accessToken.TokenableType != "admin" {
			response.Abort(ctx, http.StatusUnauthorized, "invalid_token")
			return
		}

		// 查询用户信息（一户一库时已在 Tenant 中间件切到租户库）
		var admin models.Admin
		if err := appfacades.OrmQuery(ctx).Where("id", accessToken.TokenableID).First(&admin); err != nil {
			logger.ErrorfHTTP(ctx, "JWT middleware: load admin %d: %v", accessToken.TokenableID, err)
			response.Abort(ctx, http.StatusServiceUnavailable, "service_unavailable")
			return
		}
		// First returns no error when nothing matches.
		if admin.ID == 0 {
			response.Abort(ctx, http.StatusUnauthorized, "user_not_found")
			return
		}

		// Reject disabled accounts and revoke lingering sessions (match platform/user JWT).
		if admin.Status != 1 {
			_ = tokenService.DeleteTokensByUser("admin", admin.ID)
			response.Abort(ctx, http.StatusForbidden, "account_disabled")
			return
		}

		// 更新最后使用时间，并滑动延长过期时间（每分钟最多写一次）
		ttl := time.Duration(facades.Config().GetInt("jwt.ttl", 60)) * time.Minute // 默认60分钟
		_ = tokenService.Touch(accessToken, ttl)

		// 将用户信息存储到context中，供后续中间件使用
		ctx.WithValue("admin", admin)
		ctx.WithValue("token", accessToken)

		ctx.Request().Next()
	})
}

func isExpectedUnauthorizedErr(err error) bool {
	if err == nil {
		return false
	}

	// 业务错误（如 token expired / invalid argument）属于预期内 401
	if businessErr, ok := apperrors.GetBusinessError(err); ok {
		if businessErr.Code == apperrors.ErrInvalidArgument.Code {
			return true
		}
	}

	// ORM 未找到 token 也属于预期内 401
	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "record not found")
}
