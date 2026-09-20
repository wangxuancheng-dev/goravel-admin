package commands

import (
	"context"
	"fmt"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"

	"goravel/app/models"
	"goravel/app/services"
)

type SyncDemoActivities struct{}

func (r *SyncDemoActivities) Signature() string {
	return "activity:sync-status"
}

func (r *SyncDemoActivities) Description() string {
	return "Demo: sync demo_activities status from time windows (once/daily)"
}

func (r *SyncDemoActivities) Extend() command.Extend {
	return command.Extend{
		Category: "activity",
		Flags:    []command.Flag{TenantScopeFlag()},
	}
}

func (r *SyncDemoActivities) Handle(ctx console.Context) error {
	return RunTenantScoped(ctx, func(_ *models.Tenant, bound context.Context) error {
		updated, err := services.NewDemoActivityService(bound).SyncStatuses()
		if err != nil {
			return fmt.Errorf("activity:sync-status failed: %w", err)
		}
		ctx.Info(fmt.Sprintf("demo activities status synced, updated=%d", updated))
		return nil
	})
}
