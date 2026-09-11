package admin

import (
	"github.com/goravel/framework/contracts/http"
	"github.com/spf13/cast"

	apperrors "goravel/app/errors"
	"goravel/app/http/helpers"
	adminrequests "goravel/app/http/requests/admin"
	"goravel/app/http/response"
	"goravel/app/http/trans"
	"goravel/app/jobs"
	"goravel/app/models"
	"goravel/app/services"
)

type UserController struct{}

func NewUserController() *UserController {
	return &UserController{}
}

func (c *UserController) buildUserFilters(ctx http.Context) services.UserFilters {
	return services.BuildUserFiltersFromHTTP(ctx)
}

func (c *UserController) UserService(ctx http.Context) services.UserService {
	return services.NewUserService(ctx)
}

func (c *UserController) Index(ctx http.Context) http.Response {
	page := helpers.GetIntQuery(ctx, "page", 1)
	pageSize := helpers.GetIntQuery(ctx, "page_size", 10)
	filters := c.buildUserFilters(ctx)

	list, total, err := c.UserService(ctx).GetList(filters, page, pageSize)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "user", http.StatusInternalServerError, err, nil)
	}

	return response.Success(ctx, http.Json{
		"list":      list,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (c *UserController) Show(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	user, err := c.UserService(ctx).GetByID(id)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "user", http.StatusNotFound, err, map[string]any{"id": id})
	}

	return response.Success(ctx, http.Json{
		"user": user,
	})
}

func (c *UserController) Store(ctx http.Context) http.Response {
	var req adminrequests.UserCreate
	if resp := ValidateGeneratedRequest(ctx, &req); resp != nil {
		return resp
	}

	user, err := c.UserService(ctx).Create(&req)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "user", http.StatusInternalServerError, err, map[string]any{
			"username": req.Username,
		})
	}

	return response.Success(ctx, http.Json{
		"user": user,
	})
}

func (c *UserController) Update(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")

	var req adminrequests.UserUpdate
	if resp := ValidateGeneratedRequest(ctx, &req); resp != nil {
		return resp
	}

	user, err := c.UserService(ctx).Update(id, &req)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "user", http.StatusInternalServerError, err, map[string]any{"id": id})
	}

	return response.Success(ctx, http.Json{
		"user": user,
	})
}

func (c *UserController) Destroy(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if err := c.UserService(ctx).Delete(id); err != nil {
		return HandleGeneratedServiceError(ctx, "user", http.StatusInternalServerError, err, map[string]any{"id": id})
	}

	return response.Success(ctx, "delete_success", http.Json{})
}

// UpdateBalance 更新用户余额
func (c *UserController) UpdateBalance(ctx http.Context) http.Response {
	userID := helpers.GetUintRoute(ctx, "id")
	amount := cast.ToFloat64(ctx.Request().Input("amount", "0"))
	logType := ctx.Request().Input("type", "")
	source := ctx.Request().Input("source", "manual")
	description := ctx.Request().Input("description", "")
	remark := ctx.Request().Input("remark", "")

	var sourceID *uint
	if sourceIDStr := ctx.Request().Input("source_id", ""); sourceIDStr != "" {
		id := cast.ToUint(sourceIDStr)
		sourceID = &id
	}

	var operatorID *uint
	adminID, err := helpers.GetAdminIDFromContext(ctx)
	if err == nil && adminID > 0 {
		operatorID = &adminID
	}

	if userID == 0 {
		return response.Error(ctx, http.StatusBadRequest, "user_id_required")
	}
	if amount == 0 {
		return response.Error(ctx, http.StatusBadRequest, "amount_cannot_be_zero")
	}
	if logType == "" {
		return response.Error(ctx, http.StatusBadRequest, "balance_type_required")
	}

	if err := c.UserService(ctx).UpdateBalance(userID, amount, logType, source, sourceID, description, operatorID, remark); err != nil {
		return HandleGeneratedServiceError(ctx, "user", http.StatusBadRequest, err, map[string]any{
			"user_id": userID,
		})
	}

	return response.Success(ctx, "balance_update_success", http.Json{})
}

// ResetPassword 重置用户密码（管理员操作）
func (c *UserController) ResetPassword(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")

	var resetPasswordRequest adminrequests.ResetPassword
	errors, err := ctx.Request().ValidateRequest(&resetPasswordRequest)
	if err != nil {
		return response.Error(ctx, http.StatusBadRequest, err.Error())
	}
	if errors != nil {
		return response.ValidationError(ctx, http.StatusBadRequest, "validation_failed", errors.All())
	}

	if err := c.UserService(ctx).ResetPassword(id, resetPasswordRequest.Password); err != nil {
		return HandleGeneratedServiceError(ctx, "password", http.StatusInternalServerError, err, map[string]any{
			"user_id": id,
		})
	}

	return response.Success(ctx, "password_reset_success", http.Json{})
}

// Export 导出用户列表
func (c *UserController) Export(ctx http.Context) http.Response {
	filtersMap := map[string]any{
		"username": ctx.Request().Input("username", ""),
		"nickname": ctx.Request().Input("nickname", ""),
		"email":    ctx.Request().Input("email", ""),
		"phone":    ctx.Request().Input("phone", ""),
		"order_by": ctx.Request().Input("order_by", "id:desc"),
	}
	if statusStr := ctx.Request().Input("status", ""); statusStr != "" {
		filtersMap["status"] = cast.ToUint(statusStr)
	}

	result := EnqueueAsyncExport(ctx, EnqueueAsyncExportInput{
		LockResource: "users",
		ExportType:   models.ExportTypeUsers,
		Filters:      filtersMap,
		Job:          &jobs.ExportUsers{},
	})
	if result.Unauthorized {
		return response.Error(ctx, http.StatusUnauthorized, apperrors.ErrUnauthorized.Code)
	}
	if result.Blocked {
		return response.Error(ctx, http.StatusTooManyRequests, apperrors.ErrGetLockFailed.Code)
	}
	if result.Err != nil {
		return HandleGeneratedServiceError(ctx, "export", http.StatusInternalServerError, result.Err, nil)
	}

	return response.Success(ctx, http.Json{
		"export_id": result.ExportID,
		"message":   trans.Get(ctx, "queued"),
	})
}
