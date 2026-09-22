package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"

	"goravel/app/services"
)

type FlexibleScheduleTick struct{}

func (r *FlexibleScheduleTick) Signature() string {
	return "flexible-schedule:tick"
}

func (r *FlexibleScheduleTick) Description() string {
	return "Evaluate landlord flexible_schedules cron rows for the current minute"
}

func (r *FlexibleScheduleTick) Extend() command.Extend {
	return command.Extend{Category: "schedule"}
}

func (r *FlexibleScheduleTick) Handle(ctx console.Context) error {
	n, err := services.NewFlexibleScheduleService(context.Background()).TickDue(time.Now())
	if err != nil {
		return fmt.Errorf("flexible-schedule:tick failed: %w", err)
	}
	ctx.Info(fmt.Sprintf("flexible schedules dispatched=%d", n))
	return nil
}
