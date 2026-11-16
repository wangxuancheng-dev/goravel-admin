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
				case apperrors.ErrTenantMaintenance.Code:
					// Show the notice the platform operator wrote for this tenant, if any.
					if businessErr.Message != apperrors.ErrTenantMaintenance.Message {
						response.AbortWithMessage(ctx, http.StatusServiceUnavailable, businessErr.Code, businessErr.Message)
					} else {
						response.Abort(ctx, http.StatusServiceUnavailable, businessErr.Code)
					}
				case apperrors.ErrTenantNotReady.Code:
					response.Abort(ctx, http.StatusForbidden, businessErr.Code)
				case apperrors.ErrTenantHintConflict.Code:
					response.Abort(ctx, http.StatusBadRequest, businessErr.Code)
				case apperrors.ErrTenantConnectionFailed.Code, apperrors.ErrTenantLookupFailed.Code:
					// The tenant exists (or cannot be told yet): its database or the
					// platform's is unreachable. 503 lets probes / CDNs / clients retry
					// instead of treating it as a bug (500) or a missing tenant (404).
					abortUnavailable(ctx, businessErr.Code)
				default:
					// Misconfiguration (missing credentials, corrupt password ...): keep
					// its own code so the log and the client say what is wrong.
					response.Abort(ctx, http.StatusInternalServerError, businessErr.Code)
				}
				return
			}
			abortUnavailable(ctx, apperrors.ErrTenantConnectionFailed.Code)
			return
		}
		ctx.Request().Next()
	})
}

// tenantRetryAfterSeconds is the Retry-After sent with a 503 for an unreachable tenant.
const tenantRetryAfterSeconds = "5"

func abortUnavailable(ctx http.Context, code string) {
	ctx.Response().Header("Retry-After", tenantRetryAfterSeconds)
	response.Abort(ctx, http.StatusServiceUnavailable, code)
}
