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
				Usage: "Purge object storage and local backups before force-delete",
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
	withPurge := ctx.OptionBool("with-purge")
	limit := ctx.OptionInt("limit")
	n, err := services.NewTenantAdminService().CleanupExpiredDeletedTenants(days, withPurge, limit)
	if err != nil {
		return err
	}
	ctx.Info(fmt.Sprintf("force-deleted %d soft-deleted tenant(s) older than %d day(s) (with_purge=%v)", n, days, withPurge))
	return nil
}
