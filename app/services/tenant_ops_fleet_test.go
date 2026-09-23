package services

import "testing"

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
