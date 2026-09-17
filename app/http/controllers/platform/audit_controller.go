package platform

import (
	"github.com/goravel/framework/contracts/http"

	apperrors "goravel/app/errors"
	"goravel/app/http/controllers/admin"
	"goravel/app/http/helpers"
	"goravel/app/http/response"
	"goravel/app/services"
)

type AuditController struct{}

func NewAuditController() *AuditController {
	return &AuditController{}
}

// LoginLogs lists platform console login audits.
func (c *AuditController) LoginLogs(ctx http.Context) http.Response {
	page, pageSize := helpers.PaginationFromQuery(ctx, helpers.PaginationLimits{})
	filters := services.BuildPlatformLoginLogFiltersFromHTTP(ctx)
	rows, total, err := services.ListPlatformLoginLogsPaged(filters, page, pageSize)
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "platform_login_log", http.StatusInternalServerError, err, nil)
	}
	list := make([]map[string]any, 0, len(rows))
	for i := range rows {
		list = append(list, services.PlatformLoginLogToJSON(&rows[i]))
	}
	return response.Paginate(ctx, list, total, page, pageSize)
}

// LoginLogShow returns one platform login audit row.
func (c *AuditController) LoginLogShow(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}
	row, err := services.GetPlatformLoginLogByID(id)
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "platform_login_log", http.StatusNotFound, err, map[string]any{"id": id})
	}
	return response.Success(ctx, map[string]any{
		"login_log": services.PlatformLoginLogToJSON(row),
	})
}

// OperationLogs lists platform console write audits.
func (c *AuditController) OperationLogs(ctx http.Context) http.Response {
	page, pageSize := helpers.PaginationFromQuery(ctx, helpers.PaginationLimits{})
	filters := services.BuildPlatformOperationLogFiltersFromHTTP(ctx)
	rows, total, err := services.ListPlatformOperationLogsPaged(filters, page, pageSize)
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "platform_operation_log", http.StatusInternalServerError, err, nil)
	}
	list := make([]map[string]any, 0, len(rows))
	for i := range rows {
		list = append(list, services.PlatformOperationLogToJSON(&rows[i]))
	}
	return response.Paginate(ctx, list, total, page, pageSize)
}

// OperationLogShow returns one platform operation audit row.
func (c *AuditController) OperationLogShow(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}
	row, err := services.GetPlatformOperationLogByID(id)
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "platform_operation_log", http.StatusNotFound, err, map[string]any{"id": id})
	}
	return response.Success(ctx, map[string]any{
		"operation_log": services.PlatformOperationLogToJSON(row),
	})
}
