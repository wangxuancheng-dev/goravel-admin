package admin

import (
	"github.com/goravel/framework/contracts/http"

	apperrors "goravel/app/errors"
	"goravel/app/http/helpers"
	adminrequests "goravel/app/http/requests/admin"
	"goravel/app/http/response"
	"goravel/app/models"
	"goravel/app/services"
)

type PasswordController struct{}

func NewPasswordController() *PasswordController {
	return &PasswordController{}
}

func (c *PasswordController) AdminService(ctx http.Context) services.AdminService {
	return services.NewAdminServiceImpl(ctx)
}

func (c *PasswordController) currentAdminID(ctx http.Context) (uint, http.Response) {
	adminValue := ctx.Value("admin")
	if adminValue == nil {
		return 0, response.Error(ctx, http.StatusUnauthorized, apperrors.ErrNotLoggedIn.Code)
	}
	if admin, ok := adminValue.(models.Admin); ok {
		return admin.ID, nil
	}
	if adminPtr, ok := adminValue.(*models.Admin); ok && adminPtr != nil {
		return adminPtr.ID, nil
	}
	return 0, response.Error(ctx, http.StatusUnauthorized, apperrors.ErrNotLoggedIn.Code)
}

func (c *PasswordController) UpdatePassword(ctx http.Context) http.Response {
	adminID, resp := c.currentAdminID(ctx)
	if resp != nil {
		return resp
	}

	var req adminrequests.UpdatePassword
	if resp := ValidateGeneratedRequest(ctx, &req); resp != nil {
		return resp
	}

	if err := c.AdminService(ctx).UpdateOwnPassword(adminID, req.OldPassword, req.NewPassword); err != nil {
		return HandleGeneratedServiceError(ctx, "password", http.StatusInternalServerError, err, map[string]any{
			"admin_id": adminID,
		})
	}

	return response.Success(ctx, "password_update_success")
}

// ResetPassword 重置密码（管理员操作）
func (c *PasswordController) ResetPassword(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")

	var req adminrequests.ResetPassword
	if resp := ValidateGeneratedRequest(ctx, &req); resp != nil {
		return resp
	}

	if err := c.AdminService(ctx).ResetPassword(id, req.Password); err != nil {
		return HandleGeneratedServiceError(ctx, "password", http.StatusInternalServerError, err, map[string]any{
			"admin_id": id,
		})
	}

	return response.Success(ctx, "password_reset_success")
}
