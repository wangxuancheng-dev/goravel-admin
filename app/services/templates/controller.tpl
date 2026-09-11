package admin

import (
<<if .HasExport>>
<<if .ExportAsync>>
	"goravel/app/jobs"
	"goravel/app/utils"
<<end>>
<<if not .ExportAsync>>
	"github.com/spf13/cast"
<<end>>
<<end>>

	apperrors "goravel/app/errors"
	"goravel/app/http/helpers"
	adminrequests "goravel/app/http/requests/admin"
	"goravel/app/http/response"
	"goravel/app/services"

	"github.com/goravel/framework/contracts/http"
)

type <<.ControllerName>> struct {}

func (c *<<.ControllerName>>) build<<.ModelName>>Filters(ctx http.Context) services.<<.ModelName>>Filters {
	return services.Build<<.ModelName>>FiltersFromHTTP(ctx)
}

func New<<.ControllerName>>() *<<.ControllerName>> {
	return &<<.ControllerName>>{}
}

func (c *<<.ControllerName>>) <<.ServiceName>>(ctx http.Context) services.<<.ServiceName>> {
	return services.New<<.ServiceName>>(ctx)
}

// Index lists <<.ModelName>> records.
func (c *<<.ControllerName>>) Index(ctx http.Context) http.Response {
<<if .IsTreeList>>
	if c.has<<.ModelName>>TreeSearchFilters(ctx) {
		filters := c.build<<.ModelName>>Filters(ctx)
		list, _, err := c.<<.ServiceName>>(ctx).GetList(filters, 1, 10000)
		if err != nil {
			return HandleGeneratedServiceError(ctx, "<<.ModuleName>>", http.StatusInternalServerError, err, nil)
		}
		return response.Success(ctx, http.Json{
			"list": list,
		})
	}

	tree, err := c.<<.ServiceName>>(ctx).GetTree()
	if err != nil {
		return HandleGeneratedServiceError(ctx, "<<.ModuleName>>", http.StatusInternalServerError, err, nil)
	}

	return response.Success(ctx, http.Json{
		"list": tree,
	})
<<else>>
	page := helpers.GetIntQuery(ctx, "page", 1)
	pageSize := helpers.GetIntQuery(ctx, "page_size", 10)

	filters := c.build<<.ModelName>>Filters(ctx)

	list, total, err := c.<<.ServiceName>>(ctx).GetList(filters, page, pageSize)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "<<.ModuleName>>", http.StatusInternalServerError, err, nil)
	}

	return response.Success(ctx, http.Json{
		"list":      list,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
<<end>>
}

<<if .IsTreeList>>
func (c *<<.ControllerName>>) has<<.ModelName>>TreeSearchFilters(ctx http.Context) bool {
<<range .SearchableFields>>
	<<- if or (eq .SearchUIType "daterange") (eq .SearchUIType "datetimerange")>>
	if helpers.GetTimeInputOrQueryParam(ctx, "<<.Name>>_start") != "" || helpers.GetTimeInputOrQueryParam(ctx, "<<.Name>>_end") != "" {
		return true
	}
	<<- else>>
	if ctx.Request().Input("<<.Name>>", ctx.Request().Query("<<.Name>>", "")) != "" {
		return true
	}
	<<- end>>
<<- end>>
	return false
}

<<end>>

// Show returns <<.ModelName>> details.
func (c *<<.ControllerName>>) Show(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	item, err := c.<<.ServiceName>>(ctx).GetByID(id)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "<<.ModuleName>>", http.StatusNotFound, err, map[string]any{"id": id})
	}

	return response.Success(ctx, http.Json{
		"<<.ModuleName>>": item,
	})
}

// Store creates a new <<.ModelName>>.
func (c *<<.ControllerName>>) Store(ctx http.Context) http.Response {
<<- if .HasCreate>>
	var req adminrequests.<<.RequestCreateName>>
	if resp := ValidateGeneratedRequest(ctx, &req); resp != nil {
		return resp
	}

	item, err := c.<<.ServiceName>>(ctx).Create(&req)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "<<.ModuleName>>", http.StatusInternalServerError, err, nil)
	}

	return response.Success(ctx, http.Json{
		"<<.ModuleName>>": item,
	})
<<- else>>
	return response.Error(ctx, http.StatusForbidden, "create_not_allowed")
<<end>>
}

// Update modifies an existing <<.ModelName>>.
func (c *<<.ControllerName>>) Update(ctx http.Context) http.Response {
<<- if .HasEdit>>
	id := helpers.GetUintRoute(ctx, "id")

	var req adminrequests.<<.RequestUpdateName>>
	if resp := ValidateGeneratedRequest(ctx, &req); resp != nil {
		return resp
	}

	item, err := c.<<.ServiceName>>(ctx).Update(id, &req)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "<<.ModuleName>>", http.StatusInternalServerError, err, map[string]any{"id": id})
	}

	return response.Success(ctx, http.Json{
		"<<.ModuleName>>": item,
	})
<<- else>>
	return response.Error(ctx, http.StatusForbidden, "update_not_allowed")
<<end>>
}

// Destroy deletes a <<.ModelName>>.
func (c *<<.ControllerName>>) Destroy(ctx http.Context) http.Response {
<<- if .HasDelete>>
	id := helpers.GetUintRoute(ctx, "id")
	if err := c.<<.ServiceName>>(ctx).Delete(id); err != nil {
		return HandleGeneratedServiceError(ctx, "<<.ModuleName>>", http.StatusInternalServerError, err, map[string]any{"id": id})
	}

	return response.Success(ctx, "delete_success", http.Json{})
<<- else>>
	return response.Error(ctx, http.StatusForbidden, "delete_not_allowed")
<<end>>
}

// Export exports <<.ModelName>> records.
func (c *<<.ControllerName>>) Export(ctx http.Context) http.Response {
<<- if .HasExport>>
	filters := c.build<<.ModelName>>Filters(ctx)
<<- if .ExportAsync>>
	filtersMap := utils.ExportFiltersToMap(filters)
	result := EnqueueAsyncExport(ctx, EnqueueAsyncExportInput{
		LockResource: "<<.ModuleName>>s",
		ExportType:   "<<.ModuleName>>s",
		Filters:      filtersMap,
		Job:          &jobs.Export<<.ModelName>>s{},
	})
	if result.Unauthorized {
		return response.Error(ctx, http.StatusUnauthorized, apperrors.ErrUnauthorized.Code)
	}
	if result.Blocked {
		return response.Error(ctx, http.StatusTooManyRequests, apperrors.ErrGetLockFailed.Code)
	}
	if result.Err != nil {
		return HandleGeneratedServiceError(ctx, "export", http.StatusInternalServerError, result.Err, nil)
	}

	return response.Success(ctx, http.Json{
		"export_id": result.ExportID,
		"message":   "queued",
	})
<<- else>>
	lock := helpers.AcquireExportLock(ctx, "<<.ModuleName>>s")
	if lock.Unauthorized {
		return response.Error(ctx, http.StatusUnauthorized, apperrors.ErrUnauthorized.Code)
	}
	if lock.Blocked {
		return response.Error(ctx, http.StatusTooManyRequests, apperrors.ErrGetLockFailed.Code)
	}
	adminID := lock.AdminID

	list, err := c.<<.ServiceName>>(ctx).GetAll<<.ModelName>>ForExport(filters)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "<<.ModuleName>>", http.StatusInternalServerError, err, map[string]any{
			"action":   "export_<<.ModuleName>>s",
			"admin_id": adminID,
		})
	}

	headers := []string{
		<<- range .ListFields>>
		<<- if and .ShowInList (ne .Name "operation")>>
		"<<.Name>>",
		<<- end>>
		<<- end>>
	}

	timezone := helpers.GetCurrentTimezone(ctx)
	var data [][]string
	for _, row := range list {
		r := []string{
			<<- range .ListFields>>
			<<- if and .ShowInList (ne .Name "operation")>>
			<<- if eq .Name "created_at">>
			helpers.FormatCarbonWithTimezone(row.CreatedAt, timezone),
			<<- else if eq .Name "updated_at">>
			helpers.FormatCarbonWithTimezone(row.UpdatedAt, timezone),
			<<- else if eq .GoType "time.Time">>
			helpers.FormatTimeWithTimezone(row.<<.FieldName>>, timezone),
			<<- else if eq .GoType "string">>
			row.<<.FieldName>>,
			<<- else>>
			cast.ToString(row.<<.FieldName>>),
			<<- end>>
			<<- end>>
			<<- end>>
		}
		data = append(data, r)
	}

	ctx.WithValue("export_type", "<<.ModuleName>>s")

	return response.Export(ctx, "exported", headers, data, "<<.ModuleName>>s")
<<- end>>
<<- else>>
	return response.Error(ctx, http.StatusForbidden, "forbidden")
<<end>>
}

