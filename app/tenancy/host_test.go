package tenancy

import (
	"testing"

	"github.com/goravel/framework/facades"
)

func TestNormalizeHost(t *testing.T) {
	cases := map[string]string{
		"HTTPS://CRM.Example.COM:443/path": "crm.example.com",
		"crm.example.com:8443":             "crm.example.com",
		" CRM.Example.COM. ":               "crm.example.com",
		"":                                 "",
	}
	for in, want := range cases {
		if got := NormalizeHost(in); got != want {
			t.Fatalf("NormalizeHost(%q)=%q want %q", in, got, want)
		}
	}
}

func TestRequestHostPrefersForwarded(t *testing.T) {
	if got := RequestHost("api.example.com", "crm.customer.com, other"); got != "crm.customer.com" {
		t.Fatalf("got %q", got)
	}
	if got := RequestHost("api.example.com:443", ""); got != "api.example.com" {
		t.Fatalf("got %q", got)
	}
}

func TestOriginHost(t *testing.T) {
	if got := OriginHost("https://Acme.Example.COM"); got != "acme.example.com" {
		t.Fatalf("got %q", got)
	}
	if got := OriginHost(""); got != "" {
		t.Fatalf("empty: %q", got)
	}
}

func TestPickTenantHostCodePrefersRequestThenOrigin(t *testing.T) {
	resolve := func(host string) string {
		switch host {
		case "acme.example.com":
			return "acme"
		case "crm.customer.com":
			return "acme"
		default:
			return ""
		}
	}
	if got := PickTenantHostCode("acme.example.com", "https://other.example.com", resolve); got != "acme" {
		t.Fatalf("request host: %q", got)
	}
	if got := PickTenantHostCode("api.example.com", "https://acme.example.com", resolve); got != "acme" {
		t.Fatalf("origin subdomain: %q", got)
	}
	if got := PickTenantHostCode("api.example.com", "https://crm.customer.com", resolve); got != "acme" {
		t.Fatalf("origin vanity: %q", got)
	}
	if got := PickTenantHostCode("api.example.com", "https://api.example.com", resolve); got != "" {
		t.Fatalf("same host no tenant: %q", got)
	}
}

func TestIsReservedOrPlatformHostWithBase(t *testing.T) {
	prev := facades.Config().GetString("tenancy.base_domain")
	t.Cleanup(func() {
		facades.Config().Add("tenancy.base_domain", prev)
	})
	facades.Config().Add("tenancy.base_domain", "example.com")

	if !IsSubdomainOfBase("acme.example.com") {
		t.Fatal("expected subdomain of base")
	}
	if IsSubdomainOfBase("crm.customer.com") {
		t.Fatal("vanity must not count as base subdomain")
	}
	if !IsReservedOrPlatformHost("acme.example.com") {
		t.Fatal("built-in subdomain must be reserved from vanity table")
	}
	if !IsReservedOrPlatformHost("platform.example.com") {
		t.Fatal("platform host reserved")
	}
	if IsReservedOrPlatformHost("crm.customer.com") {
		t.Fatal("customer vanity should be allowed")
	}
}
