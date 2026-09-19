package platform

import (
	"strings"
	"time"

	"github.com/goravel/framework/contracts/http"

	apperrors "goravel/app/errors"
	"goravel/app/http/controllers/admin"
	"goravel/app/http/helpers"
	"goravel/app/http/response"
	"goravel/app/models"
	"goravel/app/services"
)

// SystemLogController is a read-only landlord view of system_logs
// (platform DB by default, or a selected tenant DB via tenant_code).
type SystemLogController struct{}

func NewSystemLogController() *SystemLogController {
	return &SystemLogController{}
}

func (c *SystemLogController) bindTenantCode(ctx http.Context, code string) (*models.Tenant, http.Response) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, nil
	}
	svc := services.NewTenantConnectionService()
	tenant, err := svc.FindTenantByIDOrCode(code)
	if err != nil {
		return nil, admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusNotFound, err, map[string]any{"code": code})
	}
	if err := svc.EnsureRegistered(tenant); err != nil {
		return nil, admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusBadGateway, apperrors.ErrTenantConnectionFailed.WithError(err), map[string]any{
			"code": tenant.Code,
		})
	}
	helpers.SetTenantContext(ctx, tenant.ID, tenant.ConnectionName, tenant.Code)
	return tenant, nil
}

// Index lists system logs from platform DB, or the tenant DB when tenant_code is set.
func (c *SystemLogController) Index(ctx http.Context) http.Response {
	tenantCode := strings.TrimSpace(ctx.Request().Query("tenant_code", ""))
	tenant, errResp := c.bindTenantCode(ctx, tenantCode)
	if errResp != nil {
		return errResp
	}

	page, pageSize := helpers.PaginationFromQuery(ctx, helpers.PaginationLimits{})
	filters := services.BuildSystemLogFiltersFromHTTP(ctx)
	logs, total, err := services.NewSystemLogService(ctx).GetList(filters, page, pageSize)
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "system-log", http.StatusInternalServerError, err, nil)
	}

	scope := "platform"
	code := ""
	if tenant != nil {
		scope = "tenant"
		code = tenant.Code
	}
	return response.Success(ctx, http.Json{
		"list":        logs,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"scope":       scope,
		"tenant_code": code,
	})
}

// Show returns one system log row (same DB scope as Index).
func (c *SystemLogController) Show(ctx http.Context) http.Response {
	tenantCode := strings.TrimSpace(ctx.Request().Query("tenant_code", ""))
	if _, errResp := c.bindTenantCode(ctx, tenantCode); errResp != nil {
		return errResp
	}

	id := helpers.GetUintRoute(ctx, "id")
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}
	log, err := services.NewSystemLogService(ctx).GetByID(id)
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "system-log", http.StatusNotFound, err, map[string]any{"id": id})
	}
	return response.Success(ctx, http.Json{
		"log":         *log,
		"tenant_code": tenantCode,
	})
}

// ModuleOptions returns distinct modules for the current scope.
func (c *SystemLogController) ModuleOptions(ctx http.Context) http.Response {
	tenantCode := strings.TrimSpace(ctx.Request().Query("tenant_code", ""))
	if _, errResp := c.bindTenantCode(ctx, tenantCode); errResp != nil {
		return errResp
	}
	return response.Success(ctx, http.Json{
		"modules": services.NewSystemLogService(ctx).GetModuleOptions(),
	})
}

// SystemLogSummary returns recent error/warning counts for one tenant (detail drawer).
func (c *TenantController) SystemLogSummary(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}
	tenant, err := c.service().GetByID(id)
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusNotFound, err, map[string]any{"id": id})
	}
	connSvc := services.NewTenantConnectionService()
	if err := connSvc.EnsureRegistered(tenant); err != nil {
		return response.Success(ctx, http.Json{
			"tenant_code":       tenant.Code,
			"error_count_24h":   0,
			"warning_count_24h": 0,
			"recent":            []models.SystemLog{},
			"error":             err.Error(),
		})
	}
	helpers.SetTenantContext(ctx, tenant.ID, tenant.ConnectionName, tenant.Code)

	since := time.Now().Add(-24 * time.Hour).Format("2006-01-02 15:04:05")
	svc := services.NewSystemLogService(ctx)
	errorLogs, errorCount, _ := svc.GetList(services.SystemLogFilters{
		Level:     "error",
		StartTime: since,
		OrderBy:   "id:desc",
	}, 1, 5)
	_, warningCount, _ := svc.GetList(services.SystemLogFilters{
		Level:     "warning",
		StartTime: since,
		OrderBy:   "id:desc",
	}, 1, 1)

	return response.Success(ctx, http.Json{
		"tenant_code":       tenant.Code,
		"error_count_24h":   errorCount,
		"warning_count_24h": warningCount,
		"recent":            errorLogs,
	})
}
