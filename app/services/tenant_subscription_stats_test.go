package services

import (
	"testing"
	"time"
)

func TestBuildTenantSubscriptionStats(t *testing.T) {
	now := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	in7 := now.Add(3 * 24 * time.Hour)
	in20 := now.Add(20 * 24 * time.Hour)
	past := now.Add(-24 * time.Hour)

	tenantIDs := []uint{1, 2, 3, 4, 5}
	picks := map[uint]activeSubPick{
		1: {ID: 10, PlanCode: "free"},
		2: {ID: 11, PlanCode: "pro", EndsAt: &in7},
		3: {ID: 12, PlanCode: "pro", EndsAt: &in20},
		4: {ID: 13, PlanCode: "free", EndsAt: &past},
		// 5: no pick => no subscription
	}

	stats := BuildTenantSubscriptionStats(tenantIDs, picks, []string{"free", "pro"}, now)
	if stats.ByPlanCode["free"] != 2 {
		t.Fatalf("free=%d want 2", stats.ByPlanCode["free"])
	}
	if stats.ByPlanCode["pro"] != 2 {
		t.Fatalf("pro=%d want 2", stats.ByPlanCode["pro"])
	}
	if stats.NoSubscription != 1 {
		t.Fatalf("no_subscription=%d want 1", stats.NoSubscription)
	}
	if stats.Expiring7d != 1 {
		t.Fatalf("expiring_7d=%d want 1", stats.Expiring7d)
	}
	if stats.Expiring30d != 2 {
		t.Fatalf("expiring_30d=%d want 2", stats.Expiring30d)
	}
}
