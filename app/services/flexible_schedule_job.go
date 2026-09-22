package services

import (
	"context"
	"errors"
	"strings"

	"github.com/goravel/framework/facades"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/tenancy"
)

var (
	// ErrFlexibleScheduleJobArgs is returned when queue args are invalid.
	ErrFlexibleScheduleJobArgs = apperrors.ErrInvalidArgument.WithMessage("flexible schedule job args invalid")

	// EnqueueFlexibleScheduleRunFn dispatches a row to the schedule queue (set from queue provider).
	// When nil, EnqueueFlexibleScheduleRun executes inline (sync driver / tests).
	EnqueueFlexibleScheduleRunFn func(scheduleID uint, slot string) error
)

// FlexibleScheduleQueueName is the logical queue for per-tenant cron handlers.
func FlexibleScheduleQueueName() string {
	name := strings.TrimSpace(facades.Config().GetString("tenancy.flex_schedule_queue", "schedule"))
	if name == "" {
		return "schedule"
	}
	return name
}

// EnqueueFlexibleScheduleRun queues one execution or runs inline when no hook is registered.
func EnqueueFlexibleScheduleRun(scheduleID uint, slot string) error {
	if EnqueueFlexibleScheduleRunFn != nil {
		return EnqueueFlexibleScheduleRunFn(scheduleID, slot)
	}
	return RunFlexibleScheduleJob(context.Background(), scheduleID, slot)
}

// RunFlexibleScheduleJob loads a landlord row and runs its handler (with tenant bind when needed).
func RunFlexibleScheduleJob(ctx context.Context, scheduleID uint, slot string) error {
	row, err := loadFlexibleScheduleByID(scheduleID)
	if err != nil {
		return err
	}
	if !row.Enabled {
		return apperrors.ErrFlexibleScheduleDisabled
	}
	if tenancy.Enabled() && row.TenantID > 0 {
		var bindErr error
		ctx, bindErr = NewTenantConnectionService().BindBackground(ctx, row.TenantID)
		if bindErr != nil {
			return bindErr
		}
	}
	svc := NewFlexibleScheduleService(ctx)
	return svc.executeRow(row, slot)
}

func loadFlexibleScheduleByID(id uint) (*models.FlexibleSchedule, error) {
	q := appfacades.PlatformOrmQuery(nil)
	if q == nil {
		return nil, errors.New("platform orm unavailable")
	}
	var row models.FlexibleSchedule
	if err := q.Where("id", id).First(&row); err != nil || row.ID == 0 {
		return nil, apperrors.ErrFlexibleScheduleNotFound.WithError(err)
	}
	return &row, nil
}
