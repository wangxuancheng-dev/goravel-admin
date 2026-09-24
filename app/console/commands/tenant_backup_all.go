package commands

import (
	"fmt"
	"strings"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"
	"github.com/spf13/cast"

	"goravel/app/services"
	"goravel/app/tenancy"
)

type TenantBackupAll struct{}

func (r *TenantBackupAll) Signature() string { return "tenant:backup-all" }
func (r *TenantBackupAll) Description() string {
	return "Enqueue ready-tenant backups as tenant_ops_fleet (TENANCY_BACKUP_CONCURRENCY; --rotate for one page)"
}
func (r *TenantBackupAll) Extend() command.Extend {
	return command.Extend{
		Category: "tenant",
		Flags: []command.Flag{
			&command.IntFlag{Name: "keep", Usage: "keep last N backups (passed to tenant_ops; default TENANT_BACKUP_KEEP)"},
			&command.IntFlag{Name: "limit", Usage: "max tenants this run (0=auto/drain; -1=drain all pages into queue)"},
			&command.StringFlag{Name: "after-id", Usage: "exclusive id cursor (id > after-id)"},
			&command.BoolFlag{Name: "rotate", Usage: "one page only and persist cursor (for scheduled batches)"},
		},
	}
}

func (r *TenantBackupAll) Handle(ctx console.Context) error {
	if !tenancy.Enabled() {
		ctx.Error("requires TENANCY_DRIVER=database")
		return nil
	}
	keep := -1
	if raw := strings.TrimSpace(ctx.Option("keep")); raw != "" {
		keep = ctx.OptionInt("keep")
	}
	opts := services.TenantBackupFanOutOptions{
		Keep:      keep,
		Rotate:    ctx.OptionBool("rotate"),
		RotateKey: "tenant:backup-all",
		Actor:     services.TenantOpActor{Name: "cli:tenant:backup-all"},
	}
	if raw := strings.TrimSpace(ctx.Option("limit")); raw != "" {
		opts.Limit = ctx.OptionInt("limit")
		if raw == "-1" {
			opts.Limit = -1
		}
	}
	if raw := strings.TrimSpace(ctx.Option("after-id")); raw != "" {
		opts.AfterID = cast.ToUint(raw)
	}
	if opts.Rotate {
		ctx.Info(fmt.Sprintf("enqueue one backup page (rotate; backup_concurrency=%d)", tenancy.BackupConcurrency()))
	} else {
		ctx.Info(fmt.Sprintf("enqueue ready-tenant backups as fleet (backup_concurrency=%d)", tenancy.BackupConcurrency()))
	}
	report, err := services.FanOutTenantBackups(opts)
	if err != nil {
		ctx.Error(err.Error())
		return err
	}
	ctx.Info(fmt.Sprintf("batch_id=%s mode=%s concurrency=%d fleet_jobs=%d queued=%d skipped=%d failed=%d next_after_id=%d",
		report.BatchID, report.Mode, report.Concurrency, report.FleetJobs,
		report.Queued, report.Skipped, report.Failed, report.NextAfterID))
	if report.Failed > 0 {
		return fmt.Errorf("%d tenant backup enqueue(s) failed", report.Failed)
	}
	ctx.Success("tenant:backup-all enqueue done (workers run dumps on long-running fleet)")
	return nil
}
