package platform

import (
	"strings"

	"github.com/goravel/framework/contracts/http"

	apperrors "goravel/app/errors"
	"goravel/app/http/controllers/admin"
	"goravel/app/http/helpers"
	"goravel/app/http/response"
	"goravel/app/models"
	"goravel/app/services"
)

type tenantAdminResetBody struct {
	Password        string `json:"password" form:"password"`
	ConfirmPassword string `json:"confirm_password" form:"confirm_password"`
}

type tenantAdminUnlockBody struct {
	Username string `json:"username" form:"username"`
}

func (c *TenantController) loadTenantOrFail(ctx http.Context) (*models.Tenant, http.Response) {
	id := helpers.GetUintRoute(ctx, "id")
	if id == 0 {
		return nil, response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}
	tenant, err := c.service().GetByID(id)
	if err != nil {
		return nil, admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusNotFound, err, map[string]any{"id": id})
	}
	return tenant, nil
}

// TenantAdmins lists tenant-db admin accounts for support.
func (c *TenantController) TenantAdmins(ctx http.Context) http.Response {
	tenant, fail := c.loadTenantOrFail(ctx)
	if fail != nil {
		return fail
	}
	list, err := services.ListTenantAdmins(tenant, 200)
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusInternalServerError, err, map[string]any{"id": tenant.ID})
	}
	return response.Success(ctx, map[string]any{"list": list})
}

// ResetTenantAdminPassword resets a tenant admin password (forces change on next login).
func (c *TenantController) ResetTenantAdminPassword(ctx http.Context) http.Response {
	tenant, fail := c.loadTenantOrFail(ctx)
	if fail != nil {
		return fail
	}
	adminID := helpers.GetUintRoute(ctx, "adminId")
	if adminID == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}
	var body tenantAdminResetBody
	_ = ctx.Request().Bind(&body)
	if strings.TrimSpace(body.Password) == "" {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrInvalidArgument.Code)
	}
	if body.ConfirmPassword != "" && body.Password != body.ConfirmPassword {
		return response.Error(ctx, http.StatusBadRequest, "password_confirmation")
	}
	if err := services.ResetTenantAdminPassword(tenant, adminID, body.Password); err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusInternalServerError, err, map[string]any{
			"tenant_id": tenant.ID,
			"admin_id":  adminID,
		})
	}
	return response.Success(ctx, "reset_password_success")
}

// UnlockTenantAdmin clears login lockout for a tenant admin username.
func (c *TenantController) UnlockTenantAdmin(ctx http.Context) http.Response {
	tenant, fail := c.loadTenantOrFail(ctx)
	if fail != nil {
		return fail
	}
	var body tenantAdminUnlockBody
	_ = ctx.Request().Bind(&body)
	username := strings.TrimSpace(body.Username)
	if username == "" {
		username = strings.TrimSpace(ctx.Request().Input("username", ""))
	}
	if username == "" {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrInvalidArgument.Code)
	}
	if err := services.UnlockTenantAdminLogin(tenant, username); err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusInternalServerError, err, map[string]any{
			"tenant_id": tenant.ID,
			"username":  username,
		})
	}
	return response.Success(ctx, "unlock_success")
}

// ResetTenantAdmin2FA clears tenant admin Google Authenticator binding.
func (c *TenantController) ResetTenantAdmin2FA(ctx http.Context) http.Response {
	tenant, fail := c.loadTenantOrFail(ctx)
	if fail != nil {
		return fail
	}
	adminID := helpers.GetUintRoute(ctx, "adminId")
	if adminID == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}
	if err := services.ResetTenantAdmin2FA(tenant, adminID); err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusInternalServerError, err, map[string]any{
			"tenant_id": tenant.ID,
			"admin_id":  adminID,
		})
	}
	return response.Success(ctx, "unbind_success")
}

// AuditSummary returns a read-only login/operation snapshot from the tenant DB.
func (c *TenantController) AuditSummary(ctx http.Context) http.Response {
	tenant, fail := c.loadTenantOrFail(ctx)
	if fail != nil {
		return fail
	}
	summary := services.BuildTenantAuditSummary(tenant, 10)
	return response.Success(ctx, map[string]any{"audit": summary})
}
