package platform

import (
	"context"
	"time"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/http/controllers/admin"
	"goravel/app/http/response"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/app/tenancy"
)

type HealthController struct{}

func NewHealthController() *HealthController {
	return &HealthController{}
}

// Index returns platform console health: driver, DB ping, tenant counts.
func (c *HealthController) Index(ctx http.Context) http.Response {
	if !tenancy.Enabled() {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrTenancyDisabled.Code)
	}

	databaseOK := true
	if sqlDB, err := appfacades.Orm().Connection(appfacades.PlatformConnectionName()).DB(); err != nil {
		databaseOK = false
	} else {
		pingCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := sqlDB.PingContext(pingCtx); err != nil {
			databaseOK = false
		}
	}

	total, _ := appfacades.PlatformOrmQuery(ctx).Model(&models.Tenant{}).Count()
	active, _ := appfacades.PlatformOrmQuery(ctx).Model(&models.Tenant{}).Where("status", models.TenantStatusActive).Count()

	return response.Success(ctx, map[string]any{
		"driver":      facades.Config().GetString("tenancy.driver", "off"),
		"database_ok": databaseOK,
		"tenants": map[string]any{
			"total":  total,
			"active": active,
		},
		"cli_tips": []string{
			"go run . artisan platform:install -u <user> -p <pass>",
			"go run . artisan tenant:create --code=<code> --name=<name> --migrate",
			"go run . artisan tenant:migrate <code>",
			"go run . artisan tenant:seed <code>",
		},
	})
}

type PasswordController struct{}

func NewPasswordController() *PasswordController {
	return &PasswordController{}
}

type changePasswordBody struct {
	OldPassword     string `json:"old_password" form:"old_password"`
	NewPassword     string `json:"new_password" form:"new_password"`
	ConfirmPassword string `json:"confirm_password" form:"confirm_password"`
}

// Update changes the current platform admin password.
func (c *PasswordController) Update(ctx http.Context) http.Response {
	adminUser, ok := ctx.Value("platform_admin").(models.PlatformAdmin)
	if !ok || adminUser.ID == 0 {
		return response.Error(ctx, http.StatusUnauthorized, apperrors.ErrNotLoggedIn.Code)
	}

	var body changePasswordBody
	_ = ctx.Request().Bind(&body)
	if body.OldPassword == "" || body.NewPassword == "" {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrInvalidArgument.Code)
	}
	if body.ConfirmPassword != "" && body.ConfirmPassword != body.NewPassword {
		return response.Error(ctx, http.StatusBadRequest, "validation_failed")
	}
	if len(body.NewPassword) < 6 {
		return response.Error(ctx, http.StatusBadRequest, "validation_failed")
	}

	if err := services.ChangePlatformAdminPassword(adminUser.ID, body.OldPassword, body.NewPassword); err != nil {
		return admin.HandleGeneratedServiceError(ctx, "platform_auth", http.StatusInternalServerError, err, map[string]any{
			"admin_id": adminUser.ID,
		})
	}
	return response.Success(ctx, "password_update_success")
}
