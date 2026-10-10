package admin

import (
	"github.com/spf13/cast"

	apperrors "goravel/app/errors"
	"goravel/app/http/helpers"
	adminrequests "goravel/app/http/requests/admin"
	"goravel/app/http/response"
	"goravel/app/services"

	"github.com/goravel/framework/contracts/http"
)

type QuoteController struct{}

func (c *QuoteController) buildQuoteFilters(ctx http.Context) services.QuoteFilters {
	return services.BuildQuoteFiltersFromHTTP(ctx)
}

func NewQuoteController() *QuoteController {
	return &QuoteController{}
}

func (c *QuoteController) QuoteService(ctx http.Context) services.QuoteService {
	return services.NewQuoteService(ctx)
}

// Index lists Quote records.
func (c *QuoteController) Index(ctx http.Context) http.Response {

	page := helpers.GetIntQuery(ctx, "page", 1)
	pageSize := helpers.GetIntQuery(ctx, "page_size", 10)

	filters := c.buildQuoteFilters(ctx)

	list, total, err := c.QuoteService(ctx).GetList(filters, page, pageSize)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "quote", http.StatusInternalServerError, err, nil)
	}

	return response.Success(ctx, http.Json{
		"list":      list,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})

}

// Show returns Quote details.
func (c *QuoteController) Show(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	item, err := c.QuoteService(ctx).GetByID(id)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "quote", http.StatusNotFound, err, map[string]any{"id": id})
	}

	return response.Success(ctx, http.Json{
		"quote": item,
	})
}

// Store creates a new Quote.
func (c *QuoteController) Store(ctx http.Context) http.Response {
	var req adminrequests.QuoteCreate
	if resp := ValidateGeneratedRequest(ctx, &req); resp != nil {
		return resp
	}

	item, err := c.QuoteService(ctx).Create(&req)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "quote", http.StatusInternalServerError, err, nil)
	}

	return response.Success(ctx, http.Json{
		"quote": item,
	})
}

// Update modifies an existing Quote.
func (c *QuoteController) Update(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")

	var req adminrequests.QuoteUpdate
	if resp := ValidateGeneratedRequest(ctx, &req); resp != nil {
		return resp
	}

	item, err := c.QuoteService(ctx).Update(id, &req)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "quote", http.StatusInternalServerError, err, map[string]any{"id": id})
	}

	return response.Success(ctx, http.Json{
		"quote": item,
	})
}

// Destroy deletes a Quote.
func (c *QuoteController) Destroy(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if err := c.QuoteService(ctx).Delete(id); err != nil {
		return HandleGeneratedServiceError(ctx, "quote", http.StatusInternalServerError, err, map[string]any{"id": id})
	}

	return response.Success(ctx, "delete_success", http.Json{})
}

// Export exports Quote records.
func (c *QuoteController) Export(ctx http.Context) http.Response {
	filters := c.buildQuoteFilters(ctx)
	lock := helpers.AcquireExportLock(ctx, "quotes")
	if lock.Unauthorized {
		return response.Error(ctx, http.StatusUnauthorized, apperrors.ErrUnauthorized.Code)
	}
	if lock.Blocked {
		return response.Error(ctx, http.StatusTooManyRequests, apperrors.ErrGetLockFailed.Code)
	}
	adminID := lock.AdminID

	list, err := c.QuoteService(ctx).GetAllQuoteForExport(filters)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "quote", http.StatusInternalServerError, err, map[string]any{
			"action":   "export_quotes",
			"admin_id": adminID,
		})
	}

	headers := []string{
		"quote_no",
		"customer_name",
		"status",
		"remark",
		"created_at",
		"updated_at",
	}

	timezone := helpers.GetCurrentTimezone(ctx)
	var data [][]string
	for _, row := range list {
		r := []string{
			row.QuoteNo,
			row.CustomerName,
			cast.ToString(row.Status),
			row.Remark,
			helpers.FormatCarbonWithTimezone(row.CreatedAt, timezone),
			helpers.FormatCarbonWithTimezone(row.UpdatedAt, timezone),
		}
		data = append(data, r)
	}

	ctx.WithValue("export_type", "quotes")

	return response.Export(ctx, "exported", headers, data, "quotes")
}

// Import imports Quote records from CSV.
func (c *QuoteController) Import(ctx http.Context) http.Response {
	return response.Error(ctx, http.StatusForbidden, "forbidden")

}
