package services

import (
	"testing"

	"github.com/goravel/framework/facades"
)

func TestRunTenantOpsFleetEmpty(t *testing.T) {
	if err := RunTenantOpsFleet(TenantOpsFleetArgs{}); err != nil {
		t.Fatalf("empty fleet: %v", err)
	}
}

func TestEnqueuePreparedTenantOpsBatchEmpty(t *testing.T) {
	if err := EnqueuePreparedTenantOpsBatch(nil); err != nil {
		t.Fatalf("nil batch: %v", err)
	}
}

func TestEnqueuePreparedTenantOpsBatchSingleUsesTenantOps(t *testing.T) {
	called := false
	prev := EnqueueTenantOpsFn
	EnqueueTenantOpsFn = func(args TenantOpsArgs) error {
		called = true
		if args.TenantID != 42 {
			t.Fatalf("tenant id: %d", args.TenantID)
		}
		return nil
	}
	t.Cleanup(func() { EnqueueTenantOpsFn = prev })

	if err := EnqueuePreparedTenantOpsBatch([]TenantOpsArgs{{TenantID: 42, Op: "migrate"}}); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("expected single-item batch to use EnqueueTenantOps")
	}
}

func TestEnqueuePreparedTenantOpsBatchMultiUsesFleet(t *testing.T) {
	called := false
	prev := EnqueueTenantOpsFleetFn
	EnqueueTenantOpsFleetFn = func(args TenantOpsFleetArgs) error {
		called = true
		if len(args.Items) != 2 {
			t.Fatalf("items: %d", len(args.Items))
		}
		return nil
	}
	t.Cleanup(func() { EnqueueTenantOpsFleetFn = prev })

	if err := EnqueuePreparedTenantOpsBatch([]TenantOpsArgs{
		{TenantID: 1, Op: "migrate"},
		{TenantID: 2, Op: "migrate"},
	}); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("expected multi-item batch to use EnqueueTenantOpsFleet")
	}
}

func TestEnqueuePreparedTenantOpsBatchChunked(t *testing.T) {
	fleets := 0
	singles := 0
	prevFleet := EnqueueTenantOpsFleetFn
	prevOps := EnqueueTenantOpsFn
	EnqueueTenantOpsFleetFn = func(args TenantOpsFleetArgs) error {
		fleets++
		return nil
	}
	EnqueueTenantOpsFn = func(args TenantOpsArgs) error {
		singles++
		return nil
	}
	t.Cleanup(func() {
		EnqueueTenantOpsFleetFn = prevFleet
		EnqueueTenantOpsFn = prevOps
	})

	items := make([]TenantOpsArgs, 0, 3)
	for i := 1; i <= 3; i++ {
		items = append(items, TenantOpsArgs{TenantID: uint(i), Op: "backup"})
	}
	if err := EnqueuePreparedTenantOpsBatchChunked(items, 2); err != nil {
		t.Fatal(err)
	}
	// 2 + 1 => one fleet (2 items) + one single EnqueueTenantOps
	if fleets != 1 || singles != 1 {
		t.Fatalf("fleets=%d singles=%d want 1/1", fleets, singles)
	}
}

func TestFleetConcurrencyForOpResponseBackup(t *testing.T) {
	prev := facades.Config().GetInt("tenancy.backup_concurrency", 2)
	facades.Config().Add("tenancy.backup_concurrency", 4)
	t.Cleanup(func() { facades.Config().Add("tenancy.backup_concurrency", prev) })
	if got := FleetConcurrencyForOpResponse("backup"); got != 4 {
		t.Fatalf("got %d", got)
	}
}
