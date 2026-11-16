package commands

import (
	"fmt"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"
	"github.com/goravel/framework/facades"

	"goravel/app/services"
	"goravel/app/tenancy"
)

// TenantCleanupDeleted hard-deletes soft-deleted tenants past retention.
type TenantCleanupDeleted struct{}

func (r *TenantCleanupDeleted) Signature() string {
	return "tenant:cleanup-deleted"
}

func (r *TenantCleanupDeleted) Description() string {
	return "Hard-delete soft-deleted tenants older than TENANCY_DELETED_RETENTION_DAYS"
}

func (r *TenantCleanupDeleted) Extend() command.Extend {
	return command.Extend{
		Category: "tenant",
		Flags: []command.Flag{
			&command.IntFlag{
				Name:  "days",
				Value: -1,
				Usage: "Override retention days (-1 = config tenancy.deleted_retention_days)",
			},
			&command.BoolFlag{
				Name:  "with-purge",
				Value: true,
				Usage: "Ignored: the database, object storage and backups are always removed (kept for old scripts)",
			},
			&command.IntFlag{
				Name:  "limit",
				Value: 50,
				Usage: "Max tenants to process per run",
			},
		},
	}
}

func (r *TenantCleanupDeleted) Handle(ctx console.Context) error {
	if !tenancy.Enabled() {
		ctx.Warning("tenancy disabled; skip")
		return nil
	}
	days := ctx.OptionInt("days")
	if days < 0 {
		days = facades.Config().GetInt("tenancy.deleted_retention_days", 30)
	}
	if days <= 0 {
		ctx.Info("deleted_retention_days=0; auto cleanup disabled")
		return nil
	}
	limit := ctx.OptionInt("limit")
	n, err := services.NewTenantAdminService().CleanupExpiredDeletedTenants(days, limit)
	if err != nil {
		return err
	}
	ctx.Info(fmt.Sprintf("permanently deleted %d tenant(s) soft-deleted more than %d day(s) ago (database, storage and backups included)", n, days))
	return nil
}
