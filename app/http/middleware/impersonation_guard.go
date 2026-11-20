package middleware

import (
	"strings"

	"github.com/goravel/framework/contracts/http"

	apperrors "goravel/app/errors"
	"goravel/app/http/response"
	"goravel/app/models"
	"goravel/app/services"
)

// ImpersonationGuard blocks account-security self-service in platform impersonation
// sessions: support may act as the admin, but must not change their password or 2FA,
// or revoke their sessions.
// Must run after Jwt (which stores the token in the context).
func ImpersonationGuard() http.Middleware {
	return newMiddleware("impersonation_guard", func(ctx http.Context) {
		if isImpersonationRequest(ctx) {
			method := strings.ToUpper(ctx.Request().Method())
			path := strings.TrimSuffix(ctx.Request().Path(), "/")
			if isImpersonationBlocked(method, path) {
				response.Abort(ctx, http.StatusForbidden, apperrors.ErrImpersonationForbidden.Code)
				return
			}
		}
		ctx.Request().Next()
	})
}

func isImpersonationRequest(ctx http.Context) bool {
	switch v := ctx.Value("token").(type) {
	case *models.PersonalAccessToken:
		return v != nil && services.IsImpersonationTokenName(v.Name)
	case models.PersonalAccessToken:
		return services.IsImpersonationTokenName(v.Name)
	default:
		return false
	}
}

// isImpersonationBlocked lists the admin's own credential endpoints (not /admins/{id}/...).
func isImpersonationBlocked(method, path string) bool {
	if strings.Contains(path, "/admins/") {
		return false
	}
	switch {
	case method == http.MethodPut && strings.HasSuffix(path, "/password"):
		return true
	case method == http.MethodPost && strings.HasSuffix(path, "/google-authenticator/bind"):
		return true
	case method == http.MethodPost && strings.HasSuffix(path, "/google-authenticator/unbind"):
		return true
	// Revoking the admin's own sessions would sign the real person out of their devices.
	case method == http.MethodDelete && strings.Contains(path, "/auth/tokens/"):
		return true
	default:
		return false
	}
}
