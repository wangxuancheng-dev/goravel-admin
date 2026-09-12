package admin

import (
	"github.com/goravel/framework/contracts/http"

	apperrors "goravel/app/errors"
	"goravel/app/http/helpers"
	"goravel/app/http/response"
	"goravel/app/services"
)

// SensitiveConfirmCodeFromRequest reads confirm_code, falling back to confirm_password / code.
func SensitiveConfirmCodeFromRequest(ctx http.Context) string {
	confirmCode := ctx.Request().Input("confirm_code")
	if confirmCode == "" {
		confirmCode = ctx.Request().Input("confirm_password")
	}
	if confirmCode == "" {
		confirmCode = ctx.Request().Input("code")
	}
	return confirmCode
}

// RequireSensitiveConfirm verifies TOTP (if 2FA bound) or current password for high-risk actions.
// Returns a ready-to-send response on failure; nil when confirmation succeeds.
func RequireSensitiveConfirm(ctx http.Context, module string) http.Response {
	adminID, err := helpers.GetAdminIDFromContext(ctx)
	if err != nil {
		return response.Error(ctx, http.StatusUnauthorized, apperrors.ErrNotLoggedIn.Code)
	}
	if err := services.VerifySensitiveConfirm(ctx, adminID, SensitiveConfirmCodeFromRequest(ctx)); err != nil {
		return HandleGeneratedServiceError(ctx, module, http.StatusBadRequest, err, map[string]any{
			"admin_id": adminID,
		})
	}
	return nil
}
