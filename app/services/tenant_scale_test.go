package services

import (
	"testing"
	"time"

	"github.com/goravel/framework/facades"
)

func TestResolveScopeBatch(t *testing.T) {
	prevBatch := facades.Config().GetInt("tenancy.scope_batch", 0)
	prevAt := facades.Config().GetInt("tenancy.scope_auto_batch_at", 200)
	prevAuto := facades.Config().GetInt("tenancy.scope_auto_batch", 100)
	t.Cleanup(func() {
		facades.Config().Add("tenancy.scope_batch", prevBatch)
		facades.Config().Add("tenancy.scope_auto_batch_at", prevAt)
		facades.Config().Add("tenancy.scope_auto_batch", prevAuto)
	})

	facades.Config().Add("tenancy.scope_batch", 0)
	facades.Config().Add("tenancy.scope_auto_batch_at", 200)
	facades.Config().Add("tenancy.scope_auto_batch", 100)

	if got := ResolveScopeBatch(0, 50); got != 0 {
		t.Fatalf("small fleet want unlimited, got %d", got)
	}
	if got := ResolveScopeBatch(0, 250); got != 100 {
		t.Fatalf("large fleet auto batch want 100, got %d", got)
	}
	if got := ResolveScopeBatch(40, 250); got != 40 {
		t.Fatalf("explicit limit want 40, got %d", got)
	}
	if got := ResolveScopeBatch(-1, 250); got != 0 {
		t.Fatalf("-1 want unlimited, got %d", got)
	}

	facades.Config().Add("tenancy.scope_batch", 75)
	if got := ResolveScopeBatch(0, 10); got != 75 {
		t.Fatalf("forced batch want 75, got %d", got)
	}
}

func TestRegisteredConnEviction(t *testing.T) {
	ResetRegisteredConnsForTest()
	t.Cleanup(ResetRegisteredConnsForTest)

	prevMax := facades.Config().GetInt("tenancy.registered_max", 0)
	prevTTL := facades.Config().GetInt("tenancy.registered_idle_ttl", 900)
	t.Cleanup(func() {
		facades.Config().Add("tenancy.registered_max", prevMax)
		facades.Config().Add("tenancy.registered_idle_ttl", prevTTL)
	})

	facades.Config().Add("tenancy.registered_max", 2)
	facades.Config().Add("tenancy.registered_idle_ttl", 0)

	registeredMu.Lock()
	markRegisteredLocked("tenant_1")
	registeredAt["tenant_1"] = time.Now().Add(-time.Hour)
	markRegisteredLocked("tenant_2")
	registeredAt["tenant_2"] = time.Now().Add(-time.Minute)
	evicted := evictRegisteredLocked("tenant_3")
	registeredMu.Unlock()

	if !evicted {
		t.Fatal("expected eviction when at max")
	}
	if RegisteredConnCount() >= 2 {
		// after eviction for room, count should be < max so tenant_3 can be added
		t.Fatalf("expected count < 2 after eviction for new slot, got %d", RegisteredConnCount())
	}
	if _, ok := registeredAt["tenant_1"]; ok {
		t.Fatal("oldest tenant_1 should be evicted")
	}
}

func TestRegisteredIdleTTL(t *testing.T) {
	ResetRegisteredConnsForTest()
	t.Cleanup(ResetRegisteredConnsForTest)

	prevMax := facades.Config().GetInt("tenancy.registered_max", 0)
	prevTTL := facades.Config().GetInt("tenancy.registered_idle_ttl", 900)
	t.Cleanup(func() {
		facades.Config().Add("tenancy.registered_max", prevMax)
		facades.Config().Add("tenancy.registered_idle_ttl", prevTTL)
	})

	facades.Config().Add("tenancy.registered_max", 0)
	facades.Config().Add("tenancy.registered_idle_ttl", 60)

	registeredMu.Lock()
	markRegisteredLocked("tenant_old")
	registeredAt["tenant_old"] = time.Now().Add(-2 * time.Minute)
	markRegisteredLocked("tenant_new")
	_ = evictRegisteredLocked("")
	registeredMu.Unlock()

	if _, ok := registeredAt["tenant_old"]; ok {
		t.Fatal("idle tenant_old should be evicted")
	}
	if _, ok := registeredAt["tenant_new"]; !ok {
		t.Fatal("fresh tenant_new should remain")
	}
}

func TestComputeNextRunAt(t *testing.T) {
	from := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	next := ComputeNextRunAt("0 20 * * *", from)
	if next == nil {
		t.Fatal("expected next")
	}
	if next.Hour() != 20 || next.Minute() != 0 {
		t.Fatalf("unexpected next %v", next)
	}
	if ComputeNextRunAt("not-a-cron", from) != nil {
		t.Fatal("invalid cron should return nil")
	}
}
