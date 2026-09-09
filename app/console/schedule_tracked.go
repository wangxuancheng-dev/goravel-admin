package console

import (
	"github.com/goravel/framework/contracts/schedule"

	"goravel/app/facades"
	"goravel/app/services"
)

// ScheduleTracked registers a cron callback that runs an artisan command and records last-run info.
// Prefer this over Schedule().Command() so automatic executions appear in the admin schedule page.
// Name(command) is required for OnOneServer() locking in Goravel.
func ScheduleTracked(command string) schedule.Event {
	return facades.Schedule().Call(func() {
		_, _ = services.ExecuteScheduledCommand(command, services.ScheduleTriggerSchedule)
	}).Name(command)
}
