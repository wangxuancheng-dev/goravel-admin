package commands

import (
	"fmt"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"
	"github.com/goravel/framework/facades"

	"goravel/app/services"
	"goravel/app/tenancy"
)

// TenantBackupScheduled enqueues daily tenant backups when enabled.
// Mode full (default): every ready tenant via backup fleet (TENANCY_BACKUP_CONCURRENCY).
// Mode rotate: one page of TENANT_BACKUP_SCHEDULE_BATCH with cursor.
type TenantBackupScheduled struct{}

func (r *TenantBackupScheduled) Signature() string { return "tenant:backup-scheduled" }
func (r *TenantBackupScheduled) Description() string {
	return "Daily tenant backup enqueue (full fleet or rotate; needs TENANT_BACKUP_SCHEDULE_ENABLED=true)"
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
	mode := services.ResolveBackupScheduleMode()
	opts := services.TenantBackupFanOutOptions{
		Keep:  keep,
		Actor: services.TenantOpActor{Name: "schedule:tenant:backup-scheduled"},
	}
	if mode == services.BackupScheduleModeRotate {
		batch := services.ResolveBackupScheduleBatch()
		ctx.Info(fmt.Sprintf("enqueue up to %d tenant backups (rotate; backup_concurrency=%d)",
			batch, tenancy.BackupConcurrency()))
		opts.Limit = batch
		opts.Rotate = true
		opts.RotateKey = "tenant:backup-scheduled"
	} else {
		ctx.Info(fmt.Sprintf("enqueue all ready-tenant backups (full fleet; backup_concurrency=%d)",
			tenancy.BackupConcurrency()))
	}
	report, err := services.FanOutTenantBackups(opts)
	if err != nil {
		return err
	}
	ctx.Info(fmt.Sprintf("batch_id=%s mode=%s concurrency=%d fleet_jobs=%d queued=%d skipped=%d failed=%d next_after_id=%d",
		report.BatchID, report.Mode, report.Concurrency, report.FleetJobs,
		report.Queued, report.Skipped, report.Failed, report.NextAfterID))
	if report.Failed > 0 {
		return fmt.Errorf("%d tenant backup enqueue(s) failed", report.Failed)
	}
	ctx.Success("tenant:backup-scheduled enqueue done")
	return nil
}
