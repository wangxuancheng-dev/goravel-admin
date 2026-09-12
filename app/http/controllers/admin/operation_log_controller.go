package admin

import (
	"strconv"
	"time"

	"github.com/goravel/framework/contracts/http"

	"goravel/app/constants"
	apperrors "goravel/app/errors"
	"goravel/app/http/helpers"
	"goravel/app/http/response"
	"goravel/app/services"
)

type OperationLogController struct{}

type OperationLogBatchDestroyRequest struct {
	IDs []uint `json:"ids"`
}

func NewOperationLogController() *OperationLogController {
	return &OperationLogController{}
}

func (c *OperationLogController) OperationLogService(ctx http.Context) services.OperationLogService {
	return services.NewOperationLogService(ctx)
}

func (c *OperationLogController) Index(ctx http.Context) http.Response {
	startTimeStr := helpers.GetTimeInputOrQueryParam(ctx, "start_time")
	endTimeStr := helpers.GetTimeInputOrQueryParam(ctx, "end_time")

	if startTimeStr != "" {
		startTime, resp := parseOptionalTimeFromQuery(ctx, "start_time", "invalid_start_time")
		if resp != nil {
			return resp
		}

		endTime := time.Now().UTC()
		if endTimeStr != "" {
			parsedEndTime, endResp := parseOptionalTimeFromQuery(ctx, "end_time", "invalid_end_time")
			if endResp != nil {
				return endResp
			}
			endTime = parsedEndTime
		}

		if resp := validateTimeRangeResponse(ctx, startTime, endTime, 3); resp != nil {
			return resp
		}
	}

	filters := services.BuildOperationLogFiltersFromHTTP(ctx)
	page := helpers.GetIntQuery(ctx, "page", 1)
	pageSize := helpers.GetIntQuery(ctx, "page_size", 10)

	logs, total, err := c.OperationLogService(ctx).GetList(filters, page, pageSize)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "operation-log", http.StatusInternalServerError, err, nil)
	}

	return response.Success(ctx, http.Json{
		"list":      logs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (c *OperationLogController) Show(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	log, err := c.OperationLogService(ctx).GetByID(id, true)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "operation-log", http.StatusNotFound, err, map[string]any{"id": id})
	}
	return response.Success(ctx, http.Json{
		"log": *log,
	})
}

func (c *OperationLogController) Destroy(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if err := c.OperationLogService(ctx).Delete(id); err != nil {
		return HandleGeneratedServiceError(ctx, "operation-log", http.StatusInternalServerError, err, map[string]any{"id": id})
	}
	return response.Success(ctx, "delete_success", http.Json{})
}

func (c *OperationLogController) BatchDestroy(ctx http.Context) http.Response {
	var req OperationLogBatchDestroyRequest
	if err := ctx.Request().Bind(&req); err != nil {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrParamsError.Code)
	}
	if err := c.OperationLogService(ctx).BatchDelete(req.IDs); err != nil {
		return HandleGeneratedServiceError(ctx, "operation-log", http.StatusInternalServerError, err, map[string]any{
			"ids": req.IDs,
		})
	}
	return response.Success(ctx, "delete_success", http.Json{})
}

func (c *OperationLogController) Clean(ctx http.Context) http.Response {
	days := helpers.GetIntQuery(ctx, "days", constants.DefaultCleanLogDays)
	if err := c.OperationLogService(ctx).Clean(days); err != nil {
		return HandleGeneratedServiceError(ctx, "operation-log", http.StatusInternalServerError, err, map[string]any{
			"days": days,
		})
	}
	return response.Success(ctx, "clean_success", http.Json{})
}

// Archive exports old operation logs to CSV then deletes them.
func (c *OperationLogController) Archive(ctx http.Context) http.Response {
	days := helpers.GetIntQuery(ctx, "days", constants.DefaultCleanLogDays)
	if bodyDays := ctx.Request().Input("days"); bodyDays != "" {
		if n, err := strconv.Atoi(bodyDays); err == nil && n > 0 {
			days = n
		}
	}

	exportID, err := c.OperationLogService(ctx).Archive(days)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "operation-log", http.StatusInternalServerError, err, map[string]any{
			"days":      days,
			"export_id": exportID,
		})
	}
	return response.Success(ctx, http.Json{
		"export_id": exportID,
		"days":      days,
	})
}

func (c *OperationLogController) GetTitleOptions(ctx http.Context) http.Response {
	return response.Success(ctx, http.Json{
		"titles": c.OperationLogService(ctx).GetTitleOptions(),
	})
}
