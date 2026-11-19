package http

import (
	"strconv"
	"strings"

	contractshttp "github.com/goravel/framework/contracts/http"

	"goravel/packages/feedback"
)

// Controller exposes JSON APIs without host menu/permission coupling.
type Controller struct {
	svc *feedback.Service
}

func NewController() *Controller {
	return &Controller{svc: feedback.NewService()}
}

func (c *Controller) Index(ctx contractshttp.Context) contractshttp.Response {
	page, _ := strconv.Atoi(ctx.Request().Query("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.Request().Query("page_size", "10"))
	var statusPtr *uint8
	if raw := strings.TrimSpace(ctx.Request().Query("status", "")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n >= 0 && n <= 255 {
			v := uint8(n)
			statusPtr = &v
		}
	}
	list, total, err := c.svc.List(ctx, page, pageSize, statusPtr)
	if err != nil {
		return jsonErr(ctx, 500, err.Error())
	}
	return jsonOK(ctx, map[string]any{
		"list":      list,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (c *Controller) Show(ctx contractshttp.Context) contractshttp.Response {
	id := routeUint(ctx, "id")
	row, err := c.svc.Get(ctx, id)
	if err != nil {
		return jsonErr(ctx, 404, err.Error())
	}
	return jsonOK(ctx, map[string]any{"feedback": row})
}

func (c *Controller) Store(ctx contractshttp.Context) contractshttp.Response {
	contact := strings.TrimSpace(ctx.Request().Input("contact", ""))
	content := strings.TrimSpace(ctx.Request().Input("content", ""))
	row, err := c.svc.Create(ctx, contact, content)
	if err != nil {
		return jsonErr(ctx, 400, err.Error())
	}
	return jsonOK(ctx, map[string]any{"feedback": row})
}

func (c *Controller) MarkDone(ctx contractshttp.Context) contractshttp.Response {
	id := routeUint(ctx, "id")
	row, err := c.svc.MarkDone(ctx, id)
	if err != nil {
		return jsonErr(ctx, 404, err.Error())
	}
	return jsonOK(ctx, map[string]any{"feedback": row})
}

func (c *Controller) Destroy(ctx contractshttp.Context) contractshttp.Response {
	id := routeUint(ctx, "id")
	if err := c.svc.Delete(ctx, id); err != nil {
		return jsonErr(ctx, 404, err.Error())
	}
	return jsonOK(ctx, nil)
}

func routeUint(ctx contractshttp.Context, key string) uint {
	n, _ := strconv.ParseUint(ctx.Request().Route(key), 10, 64)
	return uint(n)
}

func jsonOK(ctx contractshttp.Context, data any) contractshttp.Response {
	payload := map[string]any{"code": 200, "message": "success"}
	if data != nil {
		payload["data"] = data
	}
	return ctx.Response().Json(contractshttp.StatusOK, payload)
}

func jsonErr(ctx contractshttp.Context, status int, message string) contractshttp.Response {
	return ctx.Response().Json(status, map[string]any{
		"code":    status,
		"message": message,
	})
}
