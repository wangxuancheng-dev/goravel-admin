package platform

import (
	"strings"

	"github.com/goravel/framework/contracts/http"

	apperrors "goravel/app/errors"
	"goravel/app/http/controllers/admin"
	"goravel/app/http/response"
	"goravel/app/services"
)

// SettingController manages landlord console settings (stored in platform_settings).
type SettingController struct{}

func NewSettingController() *SettingController {
	return &SettingController{}
}

type platformSettingsBody struct {
	CaptchaType *string `json:"captcha_type" form:"captcha_type"`
}

func platformSettingsPayload(ctx http.Context) map[string]any {
	return map[string]any{
		"captcha_type": services.GetPlatformCaptchaType(ctx),
	}
}

// Show returns current platform settings (all platform roles).
func (c *SettingController) Show(ctx http.Context) http.Response {
	return response.Success(ctx, map[string]any{"settings": platformSettingsPayload(ctx)})
}

// Update saves platform settings (owner only, enforced by route middleware).
func (c *SettingController) Update(ctx http.Context) http.Response {
	var body platformSettingsBody
	_ = ctx.Request().Bind(&body)
	if body.CaptchaType == nil {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrInvalidArgument.Code)
	}
	captchaType := strings.TrimSpace(*body.CaptchaType)
	if err := services.SetPlatformCaptchaType(ctx, captchaType); err != nil {
		if err == apperrors.ErrCaptchaTypeInvalid {
			return response.Error(ctx, http.StatusBadRequest, apperrors.ErrCaptchaTypeInvalid.Code)
		}
		return admin.HandleGeneratedServiceError(ctx, "platform_settings", http.StatusInternalServerError, err, nil)
	}
	return response.Success(ctx, map[string]any{"settings": platformSettingsPayload(ctx)})
}
