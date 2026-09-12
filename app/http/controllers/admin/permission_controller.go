package admin

import (
	"github.com/goravel/framework/contracts/http"

	"goravel/app/http/helpers"
	adminrequests "goravel/app/http/requests/admin"
	"goravel/app/http/response"
	"goravel/app/services"
)

type PermissionController struct{}

func NewPermissionController() *PermissionController {
	return &PermissionController{}
}

func (c *PermissionController) buildPermissionFilters(ctx http.Context) services.PermissionFilters {
	return services.BuildPermissionFiltersFromHTTP(ctx)
}

func (c *PermissionController) PermissionService(ctx http.Context) services.PermissionService {
	return services.NewPermissionService(ctx)
}

func (c *PermissionController) Index(ctx http.Context) http.Response {
	page := helpers.GetIntQuery(ctx, "page", 1)
	pageSize := helpers.GetIntQuery(ctx, "page_size", 10)
	filters := c.buildPermissionFilters(ctx)

	list, total, err := c.PermissionService(ctx).GetList(filters, page, pageSize)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "permission", http.StatusInternalServerError, err, nil)
	}

	return response.Success(ctx, http.Json{
		"list":      list,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (c *PermissionController) Show(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	permission, err := c.PermissionService(ctx).GetByID(id)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "permission", http.StatusNotFound, err, map[string]any{"id": id})
	}

	return response.Success(ctx, http.Json{
		"permission": permission,
	})
}

func (c *PermissionController) Store(ctx http.Context) http.Response {
	var req adminrequests.PermissionCreate
	if resp := ValidateGeneratedRequest(ctx, &req); resp != nil {
		return resp
	}

	permission, err := c.PermissionService(ctx).Create(&req)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "permission", http.StatusInternalServerError, err, map[string]any{
			"name": req.Name, "slug": req.Slug,
		})
	}

	return response.Success(ctx, http.Json{
		"permission": permission,
	})
}

func (c *PermissionController) Update(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")

	var req adminrequests.PermissionUpdate
	if resp := ValidateGeneratedRequest(ctx, &req); resp != nil {
		return resp
	}

	permission, err := c.PermissionService(ctx).Update(id, &req)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "permission", http.StatusInternalServerError, err, map[string]any{"id": id})
	}

	return response.Success(ctx, http.Json{
		"permission": permission,
	})
}

func (c *PermissionController) Destroy(ctx http.Context) http.Response {
	if resp := RequireSensitiveConfirm(ctx, "permission"); resp != nil {
		return resp
	}

	id := helpers.GetUintRoute(ctx, "id")
	if err := c.PermissionService(ctx).Delete(id); err != nil {
		return HandleGeneratedServiceError(ctx, "permission", http.StatusInternalServerError, err, map[string]any{"id": id})
	}

	return response.Success(ctx, "delete_success", http.Json{})
}
