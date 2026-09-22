package services

import (
	"encoding/json"
	"fmt"

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
	Queued      int    `json:"queued"`
	Skipped     int    `json:"skipped"`
	Failed      int    `json:"failed"`
	NextAfterID uint   `json:"next_after_id"`
	BatchID     string `json:"batch_id"`
}

// ResolveBackupScheduleBatch returns daily scheduled backup page size.
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

// FanOutTenantBackups enqueues mysqldump/pg_dump jobs for ready+active tenants.
// Does not wait for dumps: long-running workers execute TenantOps.
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
	report := &TenantBackupFanOutReport{BatchID: batchID}

	enqueuePage := func(tenants []models.Tenant) {
		for i := range tenants {
			t := &tenants[i]
			_, args, err := ops.BeginQueuedBackup(t.ID, opts.Keep, opts.Actor, batchID)
			if err != nil {
				report.Skipped++
				continue
			}
			if err := EnqueueTenantOps(args); err != nil {
				_ = ops.MarkOpFailed(t, err.Error(), args.OpLogID)
				report.Failed++
				continue
			}
			report.Queued++
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
		enqueuePage(tenants)
		next := uint(0)
		if len(tenants) >= limit {
			next = tenants[len(tenants)-1].ID
		}
		report.NextAfterID = next
		if opts.Rotate {
			storeScopeCursor(opts.RotateKey, next)
		}
		return report, nil
	}

	// Drain all ready tenants into the queue (CLI backup-all default).
	for {
		tenants, err := conn.ListReadyActiveTenantsPage(afterID, pageSize)
		if err != nil {
			return report, err
		}
		if len(tenants) == 0 {
			break
		}
		enqueuePage(tenants)
		if len(tenants) < pageSize {
			break
		}
		afterID = tenants[len(tenants)-1].ID
	}
	report.NextAfterID = 0
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
