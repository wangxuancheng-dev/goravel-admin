package commands

import (
	"fmt"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"
	"github.com/goravel/framework/facades"

	"goravel/app/services"
	"goravel/app/tenancy"
)

// TenantBackupScheduled enqueues one rotated page of tenant backups when enabled.
type TenantBackupScheduled struct{}

func (r *TenantBackupScheduled) Signature() string { return "tenant:backup-scheduled" }
func (r *TenantBackupScheduled) Description() string {
	return "Daily rotated backup enqueue (no-op unless TENANT_BACKUP_SCHEDULE_ENABLED=true)"
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
	batch := services.ResolveBackupScheduleBatch()
	ctx.Info(fmt.Sprintf("enqueue up to %d tenant backups (rotate)", batch))
	report, err := services.FanOutTenantBackups(services.TenantBackupFanOutOptions{
		Keep:      keep,
		Limit:     batch,
		Rotate:    true,
		RotateKey: "tenant:backup-scheduled",
		Actor:     services.TenantOpActor{Name: "schedule:tenant:backup-scheduled"},
	})
	if err != nil {
		return err
	}
	ctx.Info(fmt.Sprintf("batch_id=%s queued=%d skipped=%d failed=%d next_after_id=%d",
		report.BatchID, report.Queued, report.Skipped, report.Failed, report.NextAfterID))
	if report.Failed > 0 {
		return fmt.Errorf("%d tenant backup enqueue(s) failed", report.Failed)
	}
	ctx.Success("tenant:backup-scheduled enqueue done")
	return nil
}
