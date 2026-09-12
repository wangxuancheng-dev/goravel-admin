package admin

import (
	"github.com/goravel/framework/contracts/http"

	"goravel/app/http/helpers"
	adminrequests "goravel/app/http/requests/admin"
	"goravel/app/http/response"
	"goravel/app/services"
)

type DepartmentController struct{}

func NewDepartmentController() *DepartmentController {
	return &DepartmentController{}
}

func (c *DepartmentController) buildDepartmentFilters(ctx http.Context) services.DepartmentFilters {
	return services.BuildDepartmentFiltersFromHTTP(ctx)
}

func (c *DepartmentController) DepartmentService(ctx http.Context) services.DepartmentService {
	return services.NewDepartmentService(ctx)
}

func (c *DepartmentController) Index(ctx http.Context) http.Response {
	filters := c.buildDepartmentFilters(ctx)
	list, err := c.DepartmentService(ctx).GetIndex(filters)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "department", http.StatusInternalServerError, err, nil)
	}
	return response.Success(ctx, http.Json{
		"list": list,
	})
}

func (c *DepartmentController) Show(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	department, err := c.DepartmentService(ctx).GetByID(id)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "department", http.StatusNotFound, err, map[string]any{"id": id})
	}
	return response.Success(ctx, http.Json{
		"department": *department,
	})
}

func (c *DepartmentController) Store(ctx http.Context) http.Response {
	var req adminrequests.DepartmentCreate
	if resp := ValidateGeneratedRequest(ctx, &req); resp != nil {
		return resp
	}

	department, err := c.DepartmentService(ctx).Create(&req)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "department", http.StatusInternalServerError, err, map[string]any{
			"name": req.Name,
		})
	}

	return response.Success(ctx, http.Json{
		"department": department,
	})
}

func (c *DepartmentController) Update(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")

	var req adminrequests.DepartmentUpdate
	if resp := ValidateGeneratedRequest(ctx, &req); resp != nil {
		return resp
	}

	department, err := c.DepartmentService(ctx).Update(id, &req)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "department", http.StatusInternalServerError, err, map[string]any{"id": id})
	}

	return response.Success(ctx, http.Json{
		"department": *department,
	})
}

func (c *DepartmentController) Destroy(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if err := c.DepartmentService(ctx).Delete(id); err != nil {
		return HandleGeneratedServiceError(ctx, "department", http.StatusInternalServerError, err, map[string]any{"id": id})
	}
	return response.Success(ctx, "delete_success", http.Json{})
}

type DepartmentTransferAdminsRequest struct {
	ToDepartmentID uint `json:"to_department_id" form:"to_department_id"`
}

// TransferAdmins moves all admins from this department to another.
func (c *DepartmentController) TransferAdmins(ctx http.Context) http.Response {
	fromID := helpers.GetUintRoute(ctx, "id")
	var req DepartmentTransferAdminsRequest
	_ = ctx.Request().Bind(&req)
	if req.ToDepartmentID == 0 {
		req.ToDepartmentID = uint(helpers.GetIntQuery(ctx, "to_department_id", 0))
	}
	if fromID == 0 || req.ToDepartmentID == 0 {
		return response.Error(ctx, http.StatusBadRequest, "params_error")
	}

	affected, err := c.DepartmentService(ctx).TransferAdmins(fromID, req.ToDepartmentID)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "department", http.StatusInternalServerError, err, map[string]any{
			"from_department_id": fromID,
			"to_department_id":   req.ToDepartmentID,
		})
	}
	return response.Success(ctx, http.Json{
		"affected":           affected,
		"from_department_id": fromID,
		"to_department_id":   req.ToDepartmentID,
	})
}
