package admin

import (
	"github.com/goravel/framework/contracts/http"
	"github.com/spf13/cast"

	apperrors "goravel/app/errors"
	"goravel/app/http/response"
	"goravel/app/models"
	"goravel/app/services"
)

type ConfigController struct {
}

func NewConfigController() *ConfigController {
	return &ConfigController{}
}

func (r *ConfigController) ConfigService(ctx http.Context) services.ConfigService {
	return services.NewConfigService(ctx)
}

// GetByGroup 根据分组获取配置
func (r *ConfigController) GetByGroup(ctx http.Context) http.Response {
	group := ctx.Request().Route("group")
	configs, err := r.ConfigService(ctx).GetByGroup(group)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "config", http.StatusInternalServerError, err, map[string]any{
			"group": group,
		})
	}

	return response.Success(ctx, http.Json{
		"configs": configs,
	})
}

// Save 保存配置（按分组批量保存）
func (r *ConfigController) Save(ctx http.Context) http.Response {
	group := ctx.Request().Input("group")
	configsMap := ctx.Request().InputMap("configs")
	// Fallback: some clients / body parsers leave InputMap empty while All() still has configs.
	if len(configsMap) == 0 {
		if raw, ok := ctx.Request().All()["configs"].(map[string]any); ok {
			configsMap = raw
		}
	}

	if err := r.ConfigService(ctx).Save(group, configsMap); err != nil {
		return HandleGeneratedServiceError(ctx, "config", http.StatusInternalServerError, err, map[string]any{
			"group": group,
		})
	}

	return response.Success(ctx)
}

// TestEmail 测试邮件发送
func (r *ConfigController) TestEmail(ctx http.Context) http.Response {
	params := services.TestEmailParams{
		Host:       ctx.Request().Input("email_host"),
		Port:       cast.ToInt(ctx.Request().Input("email_port", "587")),
		Username:   ctx.Request().Input("email_username"),
		Password:   ctx.Request().Input("email_password"),
		From:       ctx.Request().Input("email_from"),
		FromName:   ctx.Request().Input("email_from_name"),
		Encryption: ctx.Request().Input("email_encryption", "tls"),
	}

	if params.Host == "" || params.Port == 0 || params.Username == "" || params.From == "" {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrEmailConfigRequired.Code)
	}

	adminValue := ctx.Value("admin")
	if adminValue == nil {
		return response.Error(ctx, http.StatusUnauthorized, apperrors.ErrNotLoggedIn.Code)
	}

	admin, ok := adminValue.(models.Admin)
	if !ok {
		return response.Error(ctx, http.StatusUnauthorized, apperrors.ErrNotLoggedIn.Code)
	}

	testEmail := params.From
	if admin.Email != "" {
		testEmail = admin.Email
	}

	if err := r.ConfigService(ctx).TestEmail(params, testEmail); err != nil {
		return HandleGeneratedServiceError(ctx, "config", http.StatusInternalServerError, err, map[string]any{
			"host": params.Host,
			"port": params.Port,
			"from": params.From,
			"to":   testEmail,
		})
	}

	return response.Success(ctx, "test_email_success", http.Json{
		"message": "测试邮件已发送到 " + testEmail,
	})
}
