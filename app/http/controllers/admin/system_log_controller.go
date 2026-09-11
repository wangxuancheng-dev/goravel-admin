package admin

import (
	"github.com/goravel/framework/contracts/http"

	"goravel/app/constants"
	apperrors "goravel/app/errors"
	"goravel/app/http/helpers"
	"goravel/app/http/response"
	"goravel/app/services"
)

type SystemLogController struct{}

type SystemLogBatchDestroyRequest struct {
	IDs []uint `json:"ids"`
}

func NewSystemLogController() *SystemLogController {
	return &SystemLogController{}
}

func (c *SystemLogController) SystemLogService(ctx http.Context) services.SystemLogService {
	return services.NewSystemLogService(ctx)
}

func (c *SystemLogController) Index(ctx http.Context) http.Response {
	filters := services.BuildSystemLogFiltersFromHTTP(ctx)
	page := helpers.GetIntQuery(ctx, "page", 1)
	pageSize := helpers.GetIntQuery(ctx, "page_size", 10)

	logs, total, err := c.SystemLogService(ctx).GetList(filters, page, pageSize)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "system-log", http.StatusInternalServerError, err, nil)
	}

	return response.Success(ctx, http.Json{
		"list":      logs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (c *SystemLogController) Show(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	log, err := c.SystemLogService(ctx).GetByID(id)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "system-log", http.StatusNotFound, err, map[string]any{"id": id})
	}
	return response.Success(ctx, http.Json{
		"log": *log,
	})
}

func (c *SystemLogController) Destroy(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if err := c.SystemLogService(ctx).Delete(id); err != nil {
		return HandleGeneratedServiceError(ctx, "system-log", http.StatusInternalServerError, err, map[string]any{"id": id})
	}
	return response.Success(ctx, "delete_success", http.Json{})
}

func (c *SystemLogController) BatchDestroy(ctx http.Context) http.Response {
	var req SystemLogBatchDestroyRequest
	if err := ctx.Request().Bind(&req); err != nil {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrParamsError.Code)
	}
	if err := c.SystemLogService(ctx).BatchDelete(req.IDs); err != nil {
		return HandleGeneratedServiceError(ctx, "system-log", http.StatusInternalServerError, err, map[string]any{
			"ids": req.IDs,
		})
	}
	return response.Success(ctx, "delete_success", http.Json{})
}

func (c *SystemLogController) Clean(ctx http.Context) http.Response {
	days := helpers.GetIntQuery(ctx, "days", constants.DefaultCleanLogDays)
	if err := c.SystemLogService(ctx).Clean(days); err != nil {
		return HandleGeneratedServiceError(ctx, "system-log", http.StatusInternalServerError, err, map[string]any{
			"days": days,
		})
	}
	return response.Success(ctx, "clean_success", http.Json{})
}

func (c *SystemLogController) GetModuleOptions(ctx http.Context) http.Response {
	return response.Success(ctx, http.Json{
		"modules": c.SystemLogService(ctx).GetModuleOptions(),
	})
}
