package commands

import (
	"context"
	"fmt"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"
	"github.com/goravel/framework/facades"

	"goravel/app/constants"
	"goravel/app/models"
	"goravel/app/services"
)

type ArchiveOperationLogs struct {
}

func (r *ArchiveOperationLogs) Signature() string {
	return "operation_log:archive"
}

func (r *ArchiveOperationLogs) Description() string {
	return "Archive and clean expired operation logs (export CSV then delete; tenant-scoped when tenancy is on)"
}

func (r *ArchiveOperationLogs) Extend() command.Extend {
	return command.Extend{
		Category: "operation_log",
		Flags: []command.Flag{
			TenantScopeFlag(),
			&command.IntFlag{
				Name:    "days",
				Aliases: []string{"d"},
				Value:   0,
				Usage:   "Archive operation logs older than N days (default: AUDIT_LOG_RETENTION_DAYS or 30)",
			},
		},
	}
}

func (r *ArchiveOperationLogs) Handle(ctx console.Context) error {
	days := ctx.OptionInt("days")
	if days <= 0 {
		days = facades.Config().GetInt("audit.retention_days", constants.DefaultCleanLogDays)
	}
	if days <= 0 {
		days = constants.DefaultCleanLogDays
	}

	return RunTenantScoped(ctx, func(_ *models.Tenant, bound context.Context) error {
		ctx.Info(fmt.Sprintf("Archiving operation logs older than %d days...", days))
		exportID, err := services.NewOperationLogService(bound).Archive(days)
		if err != nil {
			return fmt.Errorf("archive operation logs failed: %w", err)
		}
		ctx.Info(fmt.Sprintf("Operation log archive done, export_id=%d", exportID))
		return nil
	})
}
