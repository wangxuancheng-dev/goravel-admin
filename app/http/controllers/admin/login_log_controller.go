package admin

import (
	"github.com/goravel/framework/contracts/http"

	"goravel/app/constants"
	apperrors "goravel/app/errors"
	"goravel/app/http/helpers"
	"goravel/app/http/response"
	"goravel/app/services"
)

type LoginLogController struct{}

type LoginLogBatchDestroyRequest struct {
	IDs []uint `json:"ids"`
}

func NewLoginLogController() *LoginLogController {
	return &LoginLogController{}
}

func (c *LoginLogController) LoginLogService(ctx http.Context) services.LoginLogService {
	return services.NewLoginLogService(ctx)
}

func (c *LoginLogController) Index(ctx http.Context) http.Response {
	filters := services.BuildLoginLogFiltersFromHTTP(ctx)
	page := helpers.GetIntQuery(ctx, "page", 1)
	pageSize := helpers.GetIntQuery(ctx, "page_size", 10)

	logs, total, err := c.LoginLogService(ctx).GetList(filters, page, pageSize)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "login-log", http.StatusInternalServerError, err, nil)
	}

	return response.Success(ctx, http.Json{
		"list":      logs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (c *LoginLogController) Show(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	log, err := c.LoginLogService(ctx).GetByID(id, true)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "login-log", http.StatusNotFound, err, map[string]any{"id": id})
	}
	return response.Success(ctx, http.Json{
		"log": *log,
	})
}

func (c *LoginLogController) Destroy(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if err := c.LoginLogService(ctx).Delete(id); err != nil {
		return HandleGeneratedServiceError(ctx, "login-log", http.StatusInternalServerError, err, map[string]any{"id": id})
	}
	return response.Success(ctx, "delete_success", http.Json{})
}

func (c *LoginLogController) BatchDestroy(ctx http.Context) http.Response {
	var req LoginLogBatchDestroyRequest
	if err := ctx.Request().Bind(&req); err != nil {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrParamsError.Code)
	}
	if err := c.LoginLogService(ctx).BatchDelete(req.IDs); err != nil {
		return HandleGeneratedServiceError(ctx, "login-log", http.StatusInternalServerError, err, map[string]any{
			"ids": req.IDs,
		})
	}
	return response.Success(ctx, "delete_success", http.Json{})
}

func (c *LoginLogController) Clean(ctx http.Context) http.Response {
	days := helpers.GetIntQuery(ctx, "days", constants.DefaultCleanLogDays)
	if err := c.LoginLogService(ctx).Clean(days); err != nil {
		return HandleGeneratedServiceError(ctx, "login-log", http.StatusInternalServerError, err, map[string]any{
			"days": days,
		})
	}
	return response.Success(ctx, "clean_success", http.Json{})
}
