package admin

import (
	"strconv"
	"strings"
	"time"

	"github.com/goravel/framework/contracts/http"
	"github.com/spf13/cast"

	apperrors "goravel/app/errors"
	"goravel/app/http/response"
	"goravel/app/services"
)

type FlexibleScheduleController struct{}

func NewFlexibleScheduleController() *FlexibleScheduleController {
	return &FlexibleScheduleController{}
}

func (c *FlexibleScheduleController) svc(ctx http.Context) *services.FlexibleScheduleService {
	return services.NewFlexibleScheduleService(ctx)
}

// Handlers lists whitelist handler keys.
func (c *FlexibleScheduleController) Handlers(ctx http.Context) http.Response {
	return response.Success(ctx, http.Json{
		"list": services.ListFlexibleHandlers(),
	})
}

// Index lists flexible schedules.
func (c *FlexibleScheduleController) Index(ctx http.Context) http.Response {
	rows, err := c.svc(ctx).List()
	if err != nil {
		return HandleGeneratedServiceError(ctx, "flexible_schedule", http.StatusInternalServerError, err, nil)
	}
	list := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		list = append(list, services.FlexibleScheduleToJSON(row))
	}
	return response.Success(ctx, http.Json{
		"list":  list,
		"total": len(list),
	})
}

type flexibleScheduleBody struct {
	Name     string `json:"name" form:"name"`
	Handler  string `json:"handler" form:"handler"`
	CronExpr string `json:"cron_expr" form:"cron_expr"`
	Timezone string `json:"timezone" form:"timezone"`
	TenantID *uint  `json:"tenant_id" form:"tenant_id"`
	Enabled  *bool  `json:"enabled" form:"enabled"`
}

func (c *FlexibleScheduleController) Store(ctx http.Context) http.Response {
	var body flexibleScheduleBody
	if err := ctx.Request().Bind(&body); err != nil {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrParamsError)
	}
	in := services.FlexibleScheduleInput{
		Name:     body.Name,
		Handler:  body.Handler,
		CronExpr: body.CronExpr,
		Timezone: body.Timezone,
		Enabled:  body.Enabled,
	}
	if body.TenantID != nil {
		in.TenantID = *body.TenantID
	}
	row, err := c.svc(ctx).Create(in)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "flexible_schedule", http.StatusBadRequest, err, nil)
	}
	return response.Success(ctx, "create_success", http.Json{
		"flexible_schedule": services.FlexibleScheduleToJSON(*row),
	})
}

func (c *FlexibleScheduleController) Update(ctx http.Context) http.Response {
	id := cast.ToUint(ctx.Request().Route("id"))
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrParamsError)
	}
	var body flexibleScheduleBody
	if err := ctx.Request().Bind(&body); err != nil {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrParamsError)
	}
	in := services.FlexibleScheduleInput{
		Name:     body.Name,
		Handler:  body.Handler,
		CronExpr: body.CronExpr,
		Timezone: body.Timezone,
		Enabled:  body.Enabled,
	}
	setTenant := body.TenantID != nil
	if setTenant {
		in.TenantID = *body.TenantID
	}
	row, err := c.svc(ctx).Update(id, in, setTenant)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "flexible_schedule", http.StatusBadRequest, err, nil)
	}
	return response.Success(ctx, "update_success", http.Json{
		"flexible_schedule": services.FlexibleScheduleToJSON(*row),
	})
}

func (c *FlexibleScheduleController) Destroy(ctx http.Context) http.Response {
	id := cast.ToUint(ctx.Request().Route("id"))
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrParamsError)
	}
	if err := c.svc(ctx).Delete(id); err != nil {
		return HandleGeneratedServiceError(ctx, "flexible_schedule", http.StatusBadRequest, err, nil)
	}
	return response.Success(ctx, "delete_success", http.Json{})
}

func (c *FlexibleScheduleController) Run(ctx http.Context) http.Response {
	id := cast.ToUint(ctx.Request().Route("id"))
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrParamsError)
	}
	force := strings.EqualFold(ctx.Request().Query("force", ""), "1") ||
		strings.EqualFold(ctx.Request().Query("force", ""), "true")
	row, err := c.svc(ctx).RunNow(id, force)
	if err != nil {
		if be, ok := apperrors.GetBusinessError(err); ok && be.Code == apperrors.ErrScheduleRunFailed.Code && row != nil {
			return response.Success(ctx, "schedule_run_finished", http.Json{
				"flexible_schedule": services.FlexibleScheduleToJSON(*row),
			})
		}
		return HandleGeneratedServiceError(ctx, "flexible_schedule", http.StatusBadRequest, err, nil)
	}
	return response.Success(ctx, "schedule_run_finished", http.Json{
		"flexible_schedule": services.FlexibleScheduleToJSON(*row),
	})
}

type flexiblePreviewBody struct {
	CronExpr string `json:"cron_expr" form:"cron_expr"`
	Timezone string `json:"timezone" form:"timezone"`
	Count    int    `json:"count" form:"count"`
}

// Preview returns upcoming fire times for an expression.
func (c *FlexibleScheduleController) Preview(ctx http.Context) http.Response {
	var body flexiblePreviewBody
	_ = ctx.Request().Bind(&body)
	if strings.TrimSpace(body.CronExpr) == "" {
		body.CronExpr = ctx.Request().Query("cron_expr", "")
	}
	if strings.TrimSpace(body.Timezone) == "" {
		body.Timezone = ctx.Request().Query("timezone", "UTC")
	}
	if body.Count == 0 {
		if n, err := strconv.Atoi(ctx.Request().Query("count", "5")); err == nil {
			body.Count = n
		}
	}
	runs, err := services.PreviewNextRuns(body.CronExpr, body.Timezone, time.Now().UTC(), body.Count)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "flexible_schedule", http.StatusBadRequest, err, nil)
	}
	return response.Success(ctx, http.Json{
		"next_runs": runs,
	})
}
