package admin

import (
	"time"

	"github.com/goravel/framework/contracts/http"
	"github.com/spf13/cast"

	"goravel/app/http/helpers"
	"goravel/app/http/response"
	"goravel/app/services"
)

type UserBalanceLogController struct{}

func NewUserBalanceLogController() *UserBalanceLogController {
	return &UserBalanceLogController{}
}

func (c *UserBalanceLogController) UserBalanceLogService(ctx http.Context) services.UserBalanceLogService {
	return services.NewUserBalanceLogService(ctx)
}

func (c *UserBalanceLogController) parseTimeRangeFromQuery(ctx http.Context) (time.Time, time.Time, http.Response) {
	startTime, resp := parseOptionalTimeFromQuery(ctx, "start_time", "invalid_start_time")
	if resp != nil {
		return time.Time{}, time.Time{}, resp
	}

	endTime, resp := parseOptionalTimeFromQuery(ctx, "end_time", "invalid_end_time")
	if resp != nil {
		return time.Time{}, time.Time{}, resp
	}

	return startTime, endTime, nil
}

func (c *UserBalanceLogController) buildUserBalanceLogFilters(ctx http.Context) (services.UserBalanceLogFilters, http.Response) {
	userID := cast.ToUint(ctx.Request().Input("user_id", ctx.Request().Query("user_id", "0")))
	if userID == 0 {
		return services.UserBalanceLogFilters{}, response.Error(ctx, http.StatusBadRequest, "user_id_required_for_sharding")
	}

	startTime, endTime, resp := c.parseTimeRangeFromQuery(ctx)
	if resp != nil {
		return services.UserBalanceLogFilters{}, resp
	}

	var operatorID *uint
	if operatorIDStr := ctx.Request().Input("operator_id", ctx.Request().Query("operator_id", "")); operatorIDStr != "" {
		id := cast.ToUint(operatorIDStr)
		operatorID = &id
	}

	return services.UserBalanceLogFilters{
		UserID:     userID,
		Type:       ctx.Request().Input("type", ctx.Request().Query("type", "")),
		Source:     ctx.Request().Input("source", ctx.Request().Query("source", "")),
		Status:     ctx.Request().Input("status", ctx.Request().Query("status", "")),
		StartTime:  startTime,
		EndTime:    endTime,
		OperatorID: operatorID,
	}, nil
}

func (c *UserBalanceLogController) Index(ctx http.Context) http.Response {
	page, pageSize := helpers.PaginationFromQuery(ctx, helpers.PaginationLimits{})

	filters, resp := c.buildUserBalanceLogFilters(ctx)
	if resp != nil {
		return resp
	}

	list, total, err := c.UserBalanceLogService(ctx).GetLogs(filters, page, pageSize)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "user-balance-log", http.StatusBadRequest, err, map[string]any{
			"user_id": filters.UserID,
		})
	}

	return response.Success(ctx, http.Json{
		"list":      list,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (c *UserBalanceLogController) Statistics(ctx http.Context) http.Response {
	userID := cast.ToUint(ctx.Request().Input("user_id", ctx.Request().Query("user_id", "0")))
	if userID == 0 {
		return response.Error(ctx, http.StatusBadRequest, "user_id_required")
	}

	startTime, endTime, resp := c.parseTimeRangeFromQuery(ctx)
	if resp != nil {
		return resp
	}

	stats, err := c.UserBalanceLogService(ctx).GetUserStatistics(userID, startTime, endTime)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "user-balance-log", http.StatusBadRequest, err, map[string]any{
			"user_id": userID,
		})
	}

	return response.Success(ctx, http.Json{
		"statistics": stats,
	})
}

// Store 创建余额变动记录（手写：分表 + 可选余额回填）
func (c *UserBalanceLogController) Store(ctx http.Context) http.Response {
	userID := cast.ToUint(ctx.Request().Input("user_id", "0"))
	if userID == 0 {
		return response.Error(ctx, http.StatusBadRequest, "user_id_required_for_sharding")
	}

	logType := ctx.Request().Input("type", "")
	if logType == "" {
		return response.Error(ctx, http.StatusBadRequest, "balance_type_required")
	}

	amount := cast.ToFloat64(ctx.Request().Input("amount", "0"))
	if amount == 0 {
		return response.Error(ctx, http.StatusBadRequest, "amount_cannot_be_zero")
	}

	balance := cast.ToFloat64(ctx.Request().Input("balance", "0"))
	source := ctx.Request().Input("source", "manual")
	description := ctx.Request().Input("description", "")
	remark := ctx.Request().Input("remark", "")
	status := ctx.Request().Input("status", "success")

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

	svc := c.UserBalanceLogService(ctx)
	if balance == 0 {
		currentBalance, err := svc.GetUserBalance(userID)
		if err != nil {
			return HandleGeneratedServiceError(ctx, "user-balance-log", http.StatusInternalServerError, err, map[string]any{
				"user_id": userID,
			})
		}
		balance = currentBalance
	}

	log, err := svc.CreateLog(userID, logType, amount, balance, source, sourceID, description, operatorID, status, remark)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "user-balance-log", http.StatusBadRequest, err, map[string]any{
			"user_id": userID,
		})
	}

	return response.Success(ctx, "balance_log_create_success", http.Json{
		"data": log,
	})
}
