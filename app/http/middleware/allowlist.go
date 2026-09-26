package middleware

import (
	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"

	apperrors "goravel/app/errors"
	"goravel/app/http/helpers"
	"goravel/app/http/response"
	"goravel/app/services"
	"goravel/app/tenancy"
)

// Allowlist enforces tenant IP allowlist when enabled entries exist (empty = allow all).
func Allowlist() http.Middleware {
	return newMiddleware("allowlist", func(ctx http.Context) {
		if tenancy.Enabled() && !helpers.TenantBound(ctx) {
			ctx.Request().Next()
			return
		}
		realIP := helpers.GetRealIP(ctx)
		if err := services.EnsureClientIPAllowed(ctx, realIP); err != nil {
			if businessErr, ok := apperrors.GetBusinessError(err); ok {
				switch businessErr.Code {
				case apperrors.ErrIPNotAllowed.Code:
					facades.Log().Warningf("Allowlist middleware: IP %s not allowed", realIP)
					response.Abort(ctx, http.StatusForbidden, businessErr.Code)
					return
				}
			}
			facades.Log().Errorf("Allowlist middleware: Failed to load allowlists: %v", err)
			response.Abort(ctx, http.StatusServiceUnavailable, "service_unavailable")
			return
		}
		ctx.Request().Next()
	})
}
