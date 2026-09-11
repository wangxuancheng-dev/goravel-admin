package middleware

import (
	"github.com/goravel/framework/contracts/http"

	apperrors "goravel/app/errors"
	"goravel/app/http/response"
	"goravel/app/services"
	"goravel/app/tenancy"
)

// Tenant 在 JWT 之前解析租户并切换 ORM 连接。
// TENANCY_DRIVER=off 时直接放行（单库）。
func Tenant() http.Middleware {
	return newMiddleware("tenant", func(ctx http.Context) {
		if !tenancy.Enabled() {
			ctx.Request().Next()
			return
		}
		svc := services.NewTenantConnectionService()
		if err := svc.BindHTTP(ctx, ""); err != nil {
			if businessErr, ok := apperrors.GetBusinessError(err); ok {
				switch businessErr.Code {
				case apperrors.ErrTenantRequired.Code:
					response.Abort(ctx, http.StatusBadRequest, businessErr.Code)
				case apperrors.ErrTenantNotFound.Code:
					response.Abort(ctx, http.StatusNotFound, businessErr.Code)
				case apperrors.ErrTenantDisabled.Code:
					response.Abort(ctx, http.StatusForbidden, businessErr.Code)
				case apperrors.ErrTenantNotReady.Code:
					response.Abort(ctx, http.StatusForbidden, businessErr.Code)
				default:
					response.Abort(ctx, http.StatusInternalServerError, apperrors.ErrTenantConnectionFailed.Code)
				}
				return
			}
			response.Abort(ctx, http.StatusInternalServerError, apperrors.ErrTenantConnectionFailed.Code)
			return
		}
		ctx.Request().Next()
	})
}
