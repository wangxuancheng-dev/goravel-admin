package jobs

import (
	"context"

	"github.com/goravel/framework/facades"
	"github.com/spf13/cast"

	"goravel/app/services"
)

// FlexibleScheduleRun executes one flexible_schedules row (async worker).
type FlexibleScheduleRun struct{}

func (r *FlexibleScheduleRun) Signature() string {
	return "flexible_schedule_run"
}

func (r *FlexibleScheduleRun) Handle(args ...any) error {
	if len(args) < 2 {
		return services.ErrFlexibleScheduleJobArgs
	}
	scheduleID := cast.ToUint(args[0])
	slot := cast.ToString(args[1])
	if scheduleID == 0 {
		return services.ErrFlexibleScheduleJobArgs
	}
	if err := services.RunFlexibleScheduleJob(context.Background(), scheduleID, slot); err != nil {
		facades.Log().Errorf("FlexibleScheduleRun failed: schedule_id=%d slot=%s err=%v", scheduleID, slot, err)
		return err
	}
	return nil
}
