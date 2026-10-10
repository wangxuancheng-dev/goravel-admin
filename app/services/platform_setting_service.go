package services

import (
	"context"
	"strings"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/models"
)

// Platform setting keys (landlord DB table platform_settings).
const PlatformSettingCaptchaType = "captcha_type"

// GetPlatformSetting reads a landlord setting; returns def when missing or on any DB error
// (e.g. migration not applied yet), so login never breaks because of settings.
func GetPlatformSetting(ctx context.Context, key, def string) string {
	q := appfacades.PlatformOrmQuery(ctx)
	if q == nil {
		return def
	}
	var rows []models.PlatformSetting
	if err := q.Where("key", key).Limit(1).Get(&rows); err != nil || len(rows) == 0 {
		return def
	}
	return rows[0].Value
}

// SetPlatformSetting upserts a landlord setting.
func SetPlatformSetting(ctx context.Context, key, value string) error {
	q := appfacades.PlatformOrmQuery(ctx)
	if q == nil {
		return apperrors.ErrInvalidArgument
	}
	var rows []models.PlatformSetting
	if err := q.Where("key", key).Limit(1).Get(&rows); err != nil {
		return err
	}
	if len(rows) == 0 {
		return appfacades.PlatformOrmQuery(ctx).Create(&models.PlatformSetting{Key: key, Value: value})
	}
	_, err := appfacades.PlatformOrmQuery(ctx).Model(&rows[0]).Update("value", value)
	return err
}

// GetPlatformCaptchaType returns the platform console login captcha type (image|slide, default image).
func GetPlatformCaptchaType(ctx context.Context) string {
	return NormalizeCaptchaType(GetPlatformSetting(ctx, PlatformSettingCaptchaType, CaptchaTypeImage))
}

// SetPlatformCaptchaType validates and stores the platform console login captcha type.
func SetPlatformCaptchaType(ctx context.Context, captchaType string) error {
	t := strings.ToLower(strings.TrimSpace(captchaType))
	if t != CaptchaTypeImage && t != CaptchaTypeSlide {
		return apperrors.ErrCaptchaTypeInvalid
	}
	return SetPlatformSetting(ctx, PlatformSettingCaptchaType, t)
}
