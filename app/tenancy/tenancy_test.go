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

func TestAllowHeaderFallbackDefaults(t *testing.T) {
	prevResolver := facades.Config().GetString("tenancy.resolver")
	prevFallback := facades.Config().GetString("tenancy.allow_header_fallback")
	t.Cleanup(func() {
		facades.Config().Add("tenancy.resolver", prevResolver)
		facades.Config().Add("tenancy.allow_header_fallback", prevFallback)
	})

	facades.Config().Add("tenancy.allow_header_fallback", "")
	facades.Config().Add("tenancy.resolver", "subdomain")
	if AllowHeaderFallback() {
		t.Fatal("subdomain should deny header fallback by default")
	}
	facades.Config().Add("tenancy.resolver", "header")
	if !AllowHeaderFallback() {
		t.Fatal("header resolver should allow fallback by default")
	}
	facades.Config().Add("tenancy.resolver", "subdomain")
	facades.Config().Add("tenancy.allow_header_fallback", "true")
	if !AllowHeaderFallback() {
		t.Fatal("explicit true should allow fallback")
	}
}

func TestPaymentNotifyPath(t *testing.T) {
	prev := facades.Config().GetString("tenancy.driver")
	t.Cleanup(func() { facades.Config().Add("tenancy.driver", prev) })

	facades.Config().Add("tenancy.driver", "database")
	if got := PaymentNotifyPath("acme", "wechat"); got != "/api/payment/notify/wechat/acme" {
		t.Fatalf("got %q", got)
	}
	facades.Config().Add("tenancy.driver", "off")
	if got := PaymentNotifyPath("acme", "wechat"); got != "/api/payment/notify/wechat" {
		t.Fatalf("off got %q", got)
	}
}

func TestSubdomainHintReserved(t *testing.T) {
	prevBase := facades.Config().GetString("tenancy.base_domain")
	t.Cleanup(func() {
		facades.Config().Add("tenancy.base_domain", prevBase)
	})
	facades.Config().Add("tenancy.base_domain", "")

	if got := SubdomainHint("www.example.com"); got != "" {
		t.Fatalf("reserved www: %q", got)
	}
	if got := SubdomainHint("acme.example.com"); got != "acme" {
		t.Fatalf("tenant sub: %q", got)
	}
	if got := SubdomainHint("localhost"); got != "" {
		t.Fatalf("localhost: %q", got)
	}
}

func TestSubdomainHintRespectsBaseDomain(t *testing.T) {
	prevBase := facades.Config().GetString("tenancy.base_domain")
	t.Cleanup(func() {
		facades.Config().Add("tenancy.base_domain", prevBase)
	})
	facades.Config().Add("tenancy.base_domain", "example.com")

	if got := SubdomainHint("acme.example.com"); got != "acme" {
		t.Fatalf("apex subdomain: %q", got)
	}
	if got := SubdomainHint("crm.customer.com"); got != "" {
		t.Fatalf("vanity host must not be read as tenant code: %q", got)
	}
	if got := SubdomainHint("shop.com"); got != "" {
		t.Fatalf("apex vanity: %q", got)
	}
}

func TestMergeTenantHintsConflictAndFallback(t *testing.T) {
	got, err := MergeTenantHints("subdomain", "acme", "other", false)
	if err == nil || got != "" {
		t.Fatalf("expected conflict, got %q err=%v", got, err)
	}
	got, err = MergeTenantHints("subdomain", "acme", "ACME", false)
	if err != nil || got != "acme" {
		t.Fatalf("case-insensitive match: got %q err=%v", got, err)
	}
	got, err = MergeTenantHints("subdomain", "", "acme", false)
	if err != nil || got != "" {
		t.Fatalf("no fallback: got %q err=%v", got, err)
	}
	got, err = MergeTenantHints("subdomain", "", "acme", true)
	if err != nil || got != "acme" {
		t.Fatalf("fallback: got %q err=%v", got, err)
	}
	got, err = MergeTenantHints("header", "", "acme", false)
	if err != nil || got != "acme" {
		t.Fatalf("header resolver: got %q err=%v", got, err)
	}
}
