package tenancy

import (
	"context"
	"testing"

	"github.com/goravel/framework/facades"

	"goravel/app/tenancyctx"
)

func TestCacheKeyAndStoragePrefixUnboundFailSafe(t *testing.T) {
	prev := facades.Config().GetString("tenancy.driver")
	facades.Config().Add("tenancy.driver", "database")
	t.Cleanup(func() {
		facades.Config().Add("tenancy.driver", prev)
	})

	ctx := context.Background()
	if got := CacheKey(ctx, "lock:x"); got != "t_unbound:lock:x" {
		t.Fatalf("unbound CacheKey: %q", got)
	}
	if got := StoragePrefix(ctx); got != "tenants/_unbound_/" {
		t.Fatalf("unbound StoragePrefix: %q", got)
	}

	bound := tenancyctx.WithTenant(ctx, 9, "tenant_9", "acme")
	if got := CacheKey(bound, "lock:x"); got != "t9:lock:x" {
		t.Fatalf("bound CacheKey: %q", got)
	}
	if got := StoragePrefix(bound); got != "tenants/acme/" {
		t.Fatalf("bound StoragePrefix: %q", got)
	}
}

func TestCacheKeyOffModeUnchanged(t *testing.T) {
	prev := facades.Config().GetString("tenancy.driver")
	facades.Config().Add("tenancy.driver", "off")
	t.Cleanup(func() {
		facades.Config().Add("tenancy.driver", prev)
	})
	if got := CacheKey(context.Background(), "lock:x"); got != "lock:x" {
		t.Fatalf("off CacheKey: %q", got)
	}
	if got := StoragePrefix(context.Background()); got != "" {
		t.Fatalf("off StoragePrefix: %q", got)
	}
}
