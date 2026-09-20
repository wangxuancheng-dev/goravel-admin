package admin

import (
	"strings"

	"github.com/goravel/framework/contracts/http"

	apperrors "goravel/app/errors"
	"goravel/app/http/helpers"
	"goravel/app/http/response"
	"goravel/app/services"
)

type DemoActivityController struct{}

func NewDemoActivityController() *DemoActivityController {
	return &DemoActivityController{}
}

func (c *DemoActivityController) svc(ctx http.Context) *services.DemoActivityService {
	return services.NewDemoActivityService(ctx)
}

func (c *DemoActivityController) Index(ctx http.Context) http.Response {
	page := helpers.GetIntQuery(ctx, "page", 1)
	pageSize := helpers.GetIntQuery(ctx, "page_size", 20)
	title := strings.TrimSpace(ctx.Request().Query("title", ""))
	list, total, err := c.svc(ctx).GetList(page, pageSize, title)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "demo_activity", http.StatusInternalServerError, err, nil)
	}
	items := make([]map[string]any, 0, len(list))
	for i := range list {
		items = append(items, services.DemoActivityToJSON(&list[i]))
	}
	return response.Success(ctx, http.Json{
		"list":      items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (c *DemoActivityController) Show(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	row, err := c.svc(ctx).GetByID(id)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "demo_activity", http.StatusNotFound, err, map[string]any{"id": id})
	}
	return response.Success(ctx, http.Json{
		"demo_activity": services.DemoActivityToJSON(row),
	})
}

func (c *DemoActivityController) Store(ctx http.Context) http.Response {
	var in services.DemoActivityUpsert
	if err := ctx.Request().Bind(&in); err != nil {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrParamsError)
	}
	row, err := c.svc(ctx).Create(in)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "demo_activity", http.StatusBadRequest, err, nil)
	}
	return response.Success(ctx, "create_success", http.Json{
		"demo_activity": services.DemoActivityToJSON(row),
	})
}

func (c *DemoActivityController) Update(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	var in services.DemoActivityUpsert
	if err := ctx.Request().Bind(&in); err != nil {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrParamsError)
	}
	row, err := c.svc(ctx).Update(id, in)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "demo_activity", http.StatusBadRequest, err, map[string]any{"id": id})
	}
	return response.Success(ctx, "update_success", http.Json{
		"demo_activity": services.DemoActivityToJSON(row),
	})
}

func (c *DemoActivityController) Destroy(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if err := c.svc(ctx).Delete(id); err != nil {
		return HandleGeneratedServiceError(ctx, "demo_activity", http.StatusBadRequest, err, map[string]any{"id": id})
	}
	return response.Success(ctx, "delete_success", http.Json{})
}

func (c *DemoActivityController) CheckActive(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	payload, err := c.svc(ctx).CheckActive(id)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "demo_activity", http.StatusNotFound, err, map[string]any{"id": id})
	}
	return response.Success(ctx, payload)
}

func (c *DemoActivityController) SyncNow(ctx http.Context) http.Response {
	updated, err := c.svc(ctx).SyncStatuses()
	if err != nil {
		return HandleGeneratedServiceError(ctx, "demo_activity", http.StatusInternalServerError, err, nil)
	}
	return response.Success(ctx, http.Json{"updated": updated})
}
