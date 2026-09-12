package middleware

import (
	"strings"

	"github.com/goravel/framework/contracts/http"

	apperrors "goravel/app/errors"
	"goravel/app/http/response"
	"goravel/app/models"
)

// ForcePasswordChange 强制改密门禁：must_change_password=1 时仅允许少数自助接口。
func ForcePasswordChange() http.Middleware {
	return newMiddleware("force_password_change", func(ctx http.Context) {
		adminValue := ctx.Value("admin")
		if adminValue == nil {
			ctx.Request().Next()
			return
		}

		var mustChange uint8
		switch v := adminValue.(type) {
		case models.Admin:
			mustChange = v.MustChangePassword
		case *models.Admin:
			if v != nil {
				mustChange = v.MustChangePassword
			}
		default:
			ctx.Request().Next()
			return
		}

		if mustChange != 1 {
			ctx.Request().Next()
			return
		}

		method := strings.ToUpper(ctx.Request().Method())
		path := strings.TrimSuffix(ctx.Request().Path(), "/")
		if isForcePasswordChangeAllowed(method, path) {
			ctx.Request().Next()
			return
		}

		response.Abort(ctx, http.StatusForbidden, apperrors.ErrMustChangePassword.Code)
	})
}

func isForcePasswordChangeAllowed(method, path string) bool {
	switch {
	case method == http.MethodGet && strings.HasSuffix(path, "/info"):
		return true
	case method == http.MethodGet && strings.HasSuffix(path, "/heartbeat"):
		return true
	case method == http.MethodPost && strings.HasSuffix(path, "/logout"):
		return true
	case method == http.MethodPut && strings.HasSuffix(path, "/password") && !strings.Contains(path, "/admins/"):
		return true
	case method == http.MethodGet && strings.HasSuffix(path, "/google-authenticator/status"):
		return true
	default:
		return false
	}
}
