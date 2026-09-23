package services

import (
	"encoding/json"
	"sync"
	"sync/atomic"

	"github.com/goravel/framework/facades"

	"goravel/app/tenancy"
)

// TenantOpsFleetArgs is one long-running job that runs many TenantOpsArgs in
// parallel (same semaphore as tenant:migrate-all / seed-all).
type TenantOpsFleetArgs struct {
	Items []TenantOpsArgs `json:"items"`
	// Concurrency 0 = TENANCY_MIGRATE_CONCURRENCY.
	Concurrency int `json:"concurrency,omitempty"`
}

// EnqueueTenantOpsFleetFn dispatches a fleet job (set from queue provider).
// When nil, RunTenantOpsFleet runs inline (tests).
var EnqueueTenantOpsFleetFn func(args TenantOpsFleetArgs) error

// EnqueueTenantOpsFleet queues a fleet op or runs inline when no hook is registered.
func EnqueueTenantOpsFleet(args TenantOpsFleetArgs) error {
	if len(args.Items) == 0 {
		return nil
	}
	if len(args.Items) == 1 {
		return EnqueueTenantOps(args.Items[0])
	}
	if EnqueueTenantOpsFleetFn != nil {
		return EnqueueTenantOpsFleetFn(args)
	}
	return RunTenantOpsFleet(args)
}

// MarshalTenantOpsFleetArgsJSON encodes fleet args for the queue payload.
func MarshalTenantOpsFleetArgsJSON(args TenantOpsFleetArgs) (string, error) {
	b, err := json.Marshal(args)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// RunTenantOpsFleet executes migrate/seed items with TENANCY_MIGRATE_CONCURRENCY.
// Always returns nil after fan-out so the queue does not retry the whole fleet
// (per-tenant success/fail is already recorded on each tenant / op log).
func RunTenantOpsFleet(args TenantOpsFleetArgs) error {
	items := args.Items
	if len(items) == 0 {
		return nil
	}
	concurrency := tenancy.MigrateConcurrency()
	if args.Concurrency >= 1 {
		concurrency = tenancy.ClampMigrateConcurrency(args.Concurrency)
	}
	facades.Log().Infof("tenant_ops_fleet: items=%d concurrency=%d op=%s batch=%s",
		len(items), concurrency, items[0].Op, items[0].BatchID)

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var failed atomic.Int64

	for i := range items {
		item := items[i]
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			if err := RunTenantOp(item); err != nil {
				failed.Add(1)
				facades.Log().Errorf("tenant_ops_fleet item tenant_id=%d op=%s: %v", item.TenantID, item.Op, err)
			}
		}()
	}
	wg.Wait()

	if n := failed.Load(); n > 0 {
		facades.Log().Warningf("tenant_ops_fleet done: %d/%d item(s) returned error (batch=%s)", n, len(items), items[0].BatchID)
	} else {
		facades.Log().Infof("tenant_ops_fleet done: %d ok (batch=%s)", len(items), items[0].BatchID)
	}
	return nil
}

// EnqueuePreparedTenantOpsBatch dispatches one fleet job (or one TenantOps when n=1).
// Caller must have already BeginQueuedOp for each item.
func EnqueuePreparedTenantOpsBatch(items []TenantOpsArgs) error {
	if len(items) == 0 {
		return nil
	}
	if len(items) == 1 {
		return EnqueueTenantOps(items[0])
	}
	return EnqueueTenantOpsFleet(TenantOpsFleetArgs{Items: items})
}

// FleetConcurrencyForResponse exposes the effective concurrency for API payloads.
func FleetConcurrencyForResponse() int {
	return tenancy.MigrateConcurrency()
}