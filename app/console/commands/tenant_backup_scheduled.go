package commands

import (
	"strconv"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"
	"github.com/goravel/framework/facades"

	"goravel/app/tenancy"
)

// TenantBackupScheduled runs tenant:backup-all when TENANT_BACKUP_SCHEDULE_ENABLED=true.
type TenantBackupScheduled struct{}

func (r *TenantBackupScheduled) Signature() string { return "tenant:backup-scheduled" }
func (r *TenantBackupScheduled) Description() string {
	return "Scheduled tenant backup-all (no-op unless TENANT_BACKUP_SCHEDULE_ENABLED=true)"
}
func (r *TenantBackupScheduled) Extend() command.Extend {
	return command.Extend{Category: "tenant"}
}

func (r *TenantBackupScheduled) Handle(ctx console.Context) error {
	if !tenancy.Enabled() {
		ctx.Info("tenancy disabled; skip")
		return nil
	}
	if !facades.Config().GetBool("tenancy.backup_schedule_enabled", false) {
		ctx.Info("TENANT_BACKUP_SCHEDULE_ENABLED=false; skip")
		return nil
	}
	keep := facades.Config().GetInt("tenancy.backup_keep", 10)
	cmd := "tenant:backup-all"
	if keep > 0 {
		cmd += " --keep=" + strconv.Itoa(keep)
	}
	ctx.Info("running " + cmd)
	return facades.Artisan().Call(cmd)
}
