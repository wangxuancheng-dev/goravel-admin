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

type AdminController struct{}

func NewAdminController() *AdminController {
	return &AdminController{}
}

type createPlatformAdminBody struct {
	Username string `json:"username" form:"username"`
	Password string `json:"password" form:"password"`
	Name     string `json:"name" form:"name"`
	Role     string `json:"role" form:"role"`
}

type updatePlatformAdminBody struct {
	Name   *string `json:"name" form:"name"`
	Role   *string `json:"role" form:"role"`
	Status *uint8  `json:"status" form:"status"`
}

type resetPlatformAdminPasswordBody struct {
	Password        string `json:"password" form:"password"`
	ConfirmPassword string `json:"confirm_password" form:"confirm_password"`
}

func currentPlatformAdminID(ctx http.Context) uint {
	adminUser, ok := ctx.Value("platform_admin").(models.PlatformAdmin)
	if !ok {
		return 0
	}
	return adminUser.ID
}

// Index lists platform console admins.
func (c *AdminController) Index(ctx http.Context) http.Response {
	page, pageSize := helpers.PaginationFromQuery(ctx, helpers.PaginationLimits{})
	filters := services.PlatformAdminFilters{
		Username: ctx.Request().Query("username", ""),
		Role:     ctx.Request().Query("role", ""),
		Status:   ctx.Request().Query("status", ""),
		OrderBy:  ctx.Request().Query("order_by", ""),
	}
	rows, total, err := services.ListPlatformAdminsPaged(filters, page, pageSize)
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "platform_admin", http.StatusInternalServerError, err, nil)
	}
	list := make([]map[string]any, 0, len(rows))
	for i := range rows {
		list = append(list, services.PlatformAdminToJSON(&rows[i]))
	}
	return response.Paginate(ctx, list, total, page, pageSize)
}

// Show returns one platform admin.
func (c *AdminController) Show(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}
	row, err := services.GetPlatformAdminByID(id)
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "platform_admin", http.StatusNotFound, err, map[string]any{"id": id})
	}
	return response.Success(ctx, map[string]any{"admin": services.PlatformAdminToJSON(row)})
}

// Store creates a platform admin.
func (c *AdminController) Store(ctx http.Context) http.Response {
	var body createPlatformAdminBody
	_ = ctx.Request().Bind(&body)
	row, err := services.CreatePlatformAdmin(services.CreatePlatformAdminInput{
		Username: body.Username,
		Password: body.Password,
		Name:     body.Name,
		Role:     body.Role,
	})
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "platform_admin", http.StatusInternalServerError, err, nil)
	}
	return response.Success(ctx, map[string]any{"admin": services.PlatformAdminToJSON(row)})
}

// Update updates name/role/status.
func (c *AdminController) Update(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}
	var body updatePlatformAdminBody
	_ = ctx.Request().Bind(&body)
	row, err := services.UpdatePlatformAdmin(currentPlatformAdminID(ctx), id, services.UpdatePlatformAdminInput{
		Name:   body.Name,
		Role:   body.Role,
		Status: body.Status,
	})
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "platform_admin", http.StatusInternalServerError, err, map[string]any{"id": id})
	}
	return response.Success(ctx, map[string]any{"admin": services.PlatformAdminToJSON(row)})
}

// Destroy soft-deletes a platform admin.
func (c *AdminController) Destroy(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}
	if err := services.DeletePlatformAdmin(currentPlatformAdminID(ctx), id); err != nil {
		return admin.HandleGeneratedServiceError(ctx, "platform_admin", http.StatusInternalServerError, err, map[string]any{"id": id})
	}
	return response.Success(ctx)
}

// ResetPassword sets a new password for a platform admin.
func (c *AdminController) ResetPassword(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}
	var body resetPlatformAdminPasswordBody
	_ = ctx.Request().Bind(&body)
	if strings.TrimSpace(body.Password) == "" {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrInvalidArgument.Code)
	}
	if body.ConfirmPassword != "" && body.Password != body.ConfirmPassword {
		return response.Error(ctx, http.StatusBadRequest, "password_confirmation")
	}
	if err := services.ResetPlatformAdminPassword(id, body.Password); err != nil {
		return admin.HandleGeneratedServiceError(ctx, "platform_admin", http.StatusInternalServerError, err, map[string]any{"id": id})
	}
	return response.Success(ctx)
}
