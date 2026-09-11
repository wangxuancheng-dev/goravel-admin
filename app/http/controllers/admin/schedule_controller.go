package admin

import (
	"strings"

	"github.com/goravel/framework/contracts/http"

	apperrors "goravel/app/errors"
	"goravel/app/http/response"
	"goravel/app/services"
)

type ScheduleController struct{}

func NewScheduleController() *ScheduleController {
	return &ScheduleController{}
}

func (c *ScheduleController) scheduleService(ctx http.Context) services.ScheduleService {
	return services.NewScheduleService(ctx)
}

type scheduleRunRequest struct {
	Command string `json:"command" form:"command"`
}

// Index lists registered schedule tasks.
func (c *ScheduleController) Index(ctx http.Context) http.Response {
	tasks, err := c.scheduleService(ctx).List()
	if err != nil {
		return HandleGeneratedServiceError(ctx, "schedule", http.StatusInternalServerError, err, nil)
	}
	return response.Success(ctx, http.Json{
		"list":  tasks,
		"total": len(tasks),
	})
}

// Run manually executes a whitelisted scheduled command.
func (c *ScheduleController) Run(ctx http.Context) http.Response {
	var req scheduleRunRequest
	if err := ctx.Request().Bind(&req); err != nil {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrParamsError)
	}
	req.Command = strings.TrimSpace(req.Command)

	result, err := c.scheduleService(ctx).Run(req.Command)
	if err != nil {
		if be, ok := apperrors.GetBusinessError(err); ok {
			// Failed run still returns the result payload when available.
			if be.Code == apperrors.ErrScheduleRunFailed.Code && result != nil {
				return response.Success(ctx, "schedule_run_finished", http.Json{
					"result": result,
				})
			}
			return HandleGeneratedServiceError(ctx, "schedule", http.StatusBadRequest, be, map[string]any{
				"command": req.Command,
			})
		}
		return HandleGeneratedServiceError(ctx, "schedule", http.StatusInternalServerError, err, map[string]any{
			"command": req.Command,
			"result":  result,
		})
	}

	return response.Success(ctx, "schedule_run_finished", http.Json{
		"result": result,
	})
}
