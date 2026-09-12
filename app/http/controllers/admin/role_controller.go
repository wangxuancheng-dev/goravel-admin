package admin

import (
	"github.com/goravel/framework/contracts/http"

	"goravel/app/http/helpers"
	adminrequests "goravel/app/http/requests/admin"
	"goravel/app/http/response"
	"goravel/app/services"
)

type RoleController struct{}

func NewRoleController() *RoleController {
	return &RoleController{}
}

func (c *RoleController) buildRoleFilters(ctx http.Context) services.RoleFilters {
	return services.BuildRoleFiltersFromHTTP(ctx)
}

func (c *RoleController) RoleService(ctx http.Context) services.RoleService {
	return services.NewRoleService(ctx)
}

func (c *RoleController) Index(ctx http.Context) http.Response {
	page := helpers.GetIntQuery(ctx, "page", 1)
	pageSize := helpers.GetIntQuery(ctx, "page_size", 10)
	filters := c.buildRoleFilters(ctx)

	list, total, err := c.RoleService(ctx).GetList(filters, page, pageSize)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "role", http.StatusInternalServerError, err, nil)
	}

	return response.Success(ctx, http.Json{
		"list":      list,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (c *RoleController) Show(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	role, err := c.RoleService(ctx).GetDetail(id)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "role", http.StatusNotFound, err, map[string]any{"id": id})
	}

	return response.Success(ctx, http.Json{
		"role": role,
	})
}

func (c *RoleController) Store(ctx http.Context) http.Response {
	var req adminrequests.RoleCreate
	if resp := ValidateGeneratedRequest(ctx, &req); resp != nil {
		return resp
	}

	role, err := c.RoleService(ctx).Create(ctx, &req)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "role", http.StatusInternalServerError, err, map[string]any{
			"name": req.Name,
			"slug": req.Slug,
		})
	}

	return response.Success(ctx, http.Json{
		"role": role,
	})
}

func (c *RoleController) Update(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")

	var req adminrequests.RoleUpdate
	if resp := ValidateGeneratedRequest(ctx, &req); resp != nil {
		return resp
	}

	role, err := c.RoleService(ctx).Update(ctx, id, &req)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "role", http.StatusInternalServerError, err, map[string]any{"id": id})
	}

	return response.Success(ctx, http.Json{
		"role": role,
	})
}

func (c *RoleController) Destroy(ctx http.Context) http.Response {
	if resp := RequireSensitiveConfirm(ctx, "role"); resp != nil {
		return resp
	}

	id := helpers.GetUintRoute(ctx, "id")
	if err := c.RoleService(ctx).Delete(id); err != nil {
		return HandleGeneratedServiceError(ctx, "role", http.StatusInternalServerError, err, map[string]any{"id": id})
	}

	return response.Success(ctx, "delete_success", http.Json{})
}
