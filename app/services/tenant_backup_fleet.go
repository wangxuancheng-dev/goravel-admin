package services

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/goravel/framework/facades"

	apperrors "goravel/app/errors"
	"goravel/app/models"
	"goravel/app/tenancy"
)

// EnqueueTenantOpsFn dispatches TenantOpsArgs to the long-running queue (set from queue provider).
// When nil, EnqueueTenantOps runs the op inline (sync / tests).
var EnqueueTenantOpsFn func(args TenantOpsArgs) error

// EnqueueTenantOps queues a platform tenant op or runs inline when no hook is registered.
func EnqueueTenantOps(args TenantOpsArgs) error {
	if EnqueueTenantOpsFn != nil {
		return EnqueueTenantOpsFn(args)
	}
	return RunTenantOp(args)
}

// TenantBackupFanOutOptions controls fleet backup enqueue (CLI / scheduled).
type TenantBackupFanOutOptions struct {
	Keep int
	// Limit max tenants this call. 0 or -1 without Rotate = drain all pages into queue.
	// With Rotate, 0 uses TENANT_BACKUP_SCHEDULE_BATCH.
	Limit int
	AfterID uint
	// Rotate: one page per invocation and persist cursor (scheduled daily batches).
	Rotate bool
	RotateKey string
	Actor TenantOpActor
	BatchID string
}

// TenantBackupFanOutReport summarizes enqueue results.
type TenantBackupFanOutReport struct {
	Queued       int    `json:"queued"`
	Skipped      int    `json:"skipped"`
	Failed       int    `json:"failed"`
	NextAfterID  uint   `json:"next_after_id"`
	BatchID      string `json:"batch_id"`
	Mode         string `json:"mode"`
	Concurrency  int    `json:"concurrency"`
	FleetJobs    int    `json:"fleet_jobs"`
}

// ResolveBackupScheduleBatch returns daily scheduled backup page size (rotate mode).
func ResolveBackupScheduleBatch() int {
	n := facades.Config().GetInt("tenancy.backup_schedule_batch", 100)
	if n < 1 {
		n = 100
	}
	if n > 500 {
		n = 500
	}
	return n
}

// BackupScheduleModeFull / Rotate for TENANT_BACKUP_SCHEDULE_MODE.
const (
	BackupScheduleModeFull   = "full"
	BackupScheduleModeRotate = "rotate"
)

// ResolveBackupScheduleMode returns full (every ready tenant daily) or rotate.
func ResolveBackupScheduleMode() string {
	mode := strings.ToLower(strings.TrimSpace(facades.Config().GetString("tenancy.backup_schedule_mode", BackupScheduleModeFull)))
	if mode == BackupScheduleModeRotate {
		return BackupScheduleModeRotate
	}
	return BackupScheduleModeFull
}

// FanOutTenantBackups prepares backup ops and enqueues tenant_ops_fleet chunk(s)
// using TENANCY_BACKUP_CONCURRENCY (same parallel path as UI batch backup).
func FanOutTenantBackups(opts TenantBackupFanOutOptions) (*TenantBackupFanOutReport, error) {
	if !tenancy.Enabled() {
		return nil, apperrors.ErrTenancyDisabled
	}
	conn := NewTenantConnectionService()
	ops := NewTenantOpsService()

	batchID := opts.BatchID
	if batchID == "" {
		batchID = NewTenantOpsBatchID()
	}
	report := &TenantBackupFanOutReport{
		BatchID:     batchID,
		Mode:        "fleet",
		Concurrency: tenancy.BackupConcurrency(),
	}

	var items []TenantOpsArgs
	collectPage := func(tenants []models.Tenant) {
		for i := range tenants {
			t := &tenants[i]
			_, args, err := ops.BeginQueuedBackup(t.ID, opts.Keep, opts.Actor, batchID)
			if err != nil {
				report.Skipped++
				continue
			}
			items = append(items, args)
		}
	}

	pageSize := 100
	afterID := opts.AfterID

	// One-page mode: scheduled rotate or explicit --limit > 0.
	if opts.Rotate || opts.Limit > 0 {
		limit := opts.Limit
		if limit <= 0 {
			limit = ResolveBackupScheduleBatch()
		}
		if opts.Rotate && afterID == 0 {
			afterID = loadScopeCursor(opts.RotateKey)
		}
		tenants, err := conn.ListReadyActiveTenantsPage(afterID, limit)
		if err != nil {
			return report, err
		}
		if len(tenants) == 0 && afterID > 0 {
			tenants, err = conn.ListReadyActiveTenantsPage(0, limit)
			if err != nil {
				return report, err
			}
		}
		collectPage(tenants)
		next := uint(0)
		if len(tenants) >= limit {
			next = tenants[len(tenants)-1].ID
		}
		report.NextAfterID = next
		if opts.Rotate {
			storeScopeCursor(opts.RotateKey, next)
		}
	} else {
		// Drain all ready tenants (CLI backup-all / scheduled full).
		for {
			tenants, err := conn.ListReadyActiveTenantsPage(afterID, pageSize)
			if err != nil {
				return report, err
			}
			if len(tenants) == 0 {
				break
			}
			collectPage(tenants)
			if len(tenants) < pageSize {
				break
			}
			afterID = tenants[len(tenants)-1].ID
		}
		report.NextAfterID = 0
	}

	if len(items) == 0 {
		return report, nil
	}

	fleetJobs := (len(items) + backupFleetChunkSize - 1) / backupFleetChunkSize
	report.FleetJobs = fleetJobs
	if err := EnqueuePreparedTenantOpsBatchChunked(items, backupFleetChunkSize); err != nil {
		for _, args := range items {
			tenant, getErr := ops.admin.GetByID(args.TenantID)
			if getErr != nil {
				report.Failed++
				continue
			}
			_ = ops.MarkOpFailed(tenant, err.Error(), args.OpLogID)
			report.Failed++
		}
		return report, err
	}
	report.Queued = len(items)
	return report, nil
}

// MarshalTenantOpsArgsJSON is a helper for queue payload encoding.
func MarshalTenantOpsArgsJSON(args TenantOpsArgs) (string, error) {
	b, err := json.Marshal(args)
	if err != nil {
		return "", fmt.Errorf("marshal tenant ops: %w", err)
	}
	return string(b), nil
}
