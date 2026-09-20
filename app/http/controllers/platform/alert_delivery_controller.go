package platform

import (
	"strings"

	"github.com/goravel/framework/contracts/http"

	apperrors "goravel/app/errors"
	"goravel/app/http/controllers/admin"
	"goravel/app/http/helpers"
	"goravel/app/http/response"
	"goravel/app/services"
)

type AlertDeliveryController struct{}

func NewAlertDeliveryController() *AlertDeliveryController {
	return &AlertDeliveryController{}
}

// Index lists platform alert delivery records.
func (c *AlertDeliveryController) Index(ctx http.Context) http.Response {
	page, pageSize := helpers.PaginationFromQuery(ctx, helpers.PaginationLimits{})
	filters := services.PlatformAlertDeliveryFilters{
		Event:      ctx.Request().Query("event", ""),
		Status:     ctx.Request().Query("status", ""),
		Channel:    ctx.Request().Query("channel", ""),
		TenantCode: ctx.Request().Query("tenant_code", ""),
	}
	list, total, err := services.ListPlatformAlertDeliveries(filters, page, pageSize)
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "platform_alert", http.StatusInternalServerError, err, nil)
	}
	rows := make([]map[string]any, 0, len(list))
	for i := range list {
		rows = append(rows, services.PlatformAlertDeliveryToJSON(&list[i]))
	}
	return response.Paginate(ctx, rows, total, page, pageSize)
}

// Retry re-sends a failed webhook delivery.
func (c *AlertDeliveryController) Retry(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}
	row, err := services.RetryPlatformAlertDelivery(id)
	if err != nil && row == nil {
		return admin.HandleGeneratedServiceError(ctx, "platform_alert", http.StatusInternalServerError, err, map[string]any{"id": id})
	}
	data := map[string]any{"delivery": services.PlatformAlertDeliveryToJSON(row)}
	if err != nil {
		msg := strings.TrimSpace(err.Error())
		if msg == "" {
			msg = "retry_failed"
		}
		return response.Error(ctx, http.StatusBadGateway, msg)
	}
	return response.Success(ctx, data)
}
