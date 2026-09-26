package admin

import (
	"github.com/goravel/framework/contracts/http"

	"goravel/app/http/helpers"
	adminrequests "goravel/app/http/requests/admin"
	"goravel/app/http/response"
	"goravel/app/services"
)

type AllowlistController struct{}

func NewAllowlistController() *AllowlistController {
	return &AllowlistController{}
}

func (c *AllowlistController) AllowlistService(ctx http.Context) services.AllowlistService {
	return services.NewAllowlistService(ctx)
}

func (c *AllowlistController) Index(ctx http.Context) http.Response {
	page, pageSize := helpers.PaginationFromQuery(ctx, helpers.PaginationLimits{})
	filters := services.BuildAllowlistFiltersFromHTTP(ctx)
	list, total, err := c.AllowlistService(ctx).GetList(filters, page, pageSize)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "allowlist", http.StatusInternalServerError, err, nil)
	}
	return response.Success(ctx, http.Json{
		"list":      list,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"client_ip": helpers.GetRealIP(ctx),
	})
}

func (c *AllowlistController) Show(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	row, err := c.AllowlistService(ctx).GetByID(id)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "allowlist", http.StatusNotFound, err, map[string]any{"id": id})
	}
	return response.Success(ctx, http.Json{"allowlist": row, "client_ip": helpers.GetRealIP(ctx)})
}

func (c *AllowlistController) Store(ctx http.Context) http.Response {
	if resp := RequireSensitiveConfirm(ctx, "allowlist"); resp != nil {
		return resp
	}
	var req adminrequests.AllowlistCreate
	if resp := ValidateGeneratedRequest(ctx, &req); resp != nil {
		return resp
	}
	row, err := c.AllowlistService(ctx).Create(&req, helpers.GetRealIP(ctx))
	if err != nil {
		return HandleGeneratedServiceError(ctx, "allowlist", http.StatusBadRequest, err, map[string]any{"ip": req.IP})
	}
	return response.Success(ctx, http.Json{"allowlist": row})
}

func (c *AllowlistController) Update(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	var req adminrequests.AllowlistUpdate
	if resp := ValidateGeneratedRequest(ctx, &req); resp != nil {
		return resp
	}
	row, err := c.AllowlistService(ctx).Update(id, &req, helpers.GetRealIP(ctx))
	if err != nil {
		return HandleGeneratedServiceError(ctx, "allowlist", http.StatusBadRequest, err, map[string]any{"id": id})
	}
	return response.Success(ctx, http.Json{"allowlist": row})
}

func (c *AllowlistController) Destroy(ctx http.Context) http.Response {
	if resp := RequireSensitiveConfirm(ctx, "allowlist"); resp != nil {
		return resp
	}
	id := helpers.GetUintRoute(ctx, "id")
	if err := c.AllowlistService(ctx).Delete(id, helpers.GetRealIP(ctx)); err != nil {
		return HandleGeneratedServiceError(ctx, "allowlist", http.StatusBadRequest, err, map[string]any{"id": id})
	}
	return response.Success(ctx, "delete_success", http.Json{})
}
