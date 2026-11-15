package services

import (
	"errors"
	"testing"
	"time"

	"github.com/goravel/framework/facades"

	"goravel/app/tenantdb"
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

// Eviction itself is covered in app/tenantdb; these check the config wiring.
func TestRegisteredLimitsFromConfig(t *testing.T) {
	prevMax := facades.Config().GetInt("tenancy.registered_max", 0)
	prevTTL := facades.Config().GetInt("tenancy.registered_idle_ttl", 900)
	t.Cleanup(func() {
		facades.Config().Add("tenancy.registered_max", prevMax)
		facades.Config().Add("tenancy.registered_idle_ttl", prevTTL)
	})

	facades.Config().Add("tenancy.registered_max", 2)
	facades.Config().Add("tenancy.registered_idle_ttl", 60)
	if registeredMax() != 2 || registeredIdleTTL() != time.Minute {
		t.Fatalf("max=%d ttl=%v", registeredMax(), registeredIdleTTL())
	}
	facades.Config().Add("tenancy.registered_idle_ttl", 0)
	if registeredIdleTTL() != 0 {
		t.Fatalf("ttl 0 must disable idle eviction, got %v", registeredIdleTTL())
	}
}

func TestTenantConnectionNameParsing(t *testing.T) {
	for name, want := range map[string]uint{"tenant_12": 12, "tenant_1": 1} {
		if id, ok := tenantIDFromConnectionName(name); !ok || id != want {
			t.Fatalf("%s -> %d %v", name, id, ok)
		}
	}
	// Platform and maintenance connections are not tenant connections: the resolver
	// must reject them without querying the tenants table.
	for _, name := range []string{"mysql", "postgres", "tenant_", "tenant_0", "tenant_12_maint", "tenant_maint_acme", "tenant_x"} {
		if _, ok := tenantIDFromConnectionName(name); ok {
			t.Fatalf("%s must not parse as a tenant connection", name)
		}
		if _, err := resolveTenantConnection(name); !errors.Is(err, tenantdb.ErrUnknownConnection) {
			t.Fatalf("resolve %s: %v, want ErrUnknownConnection", name, err)
		}
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
