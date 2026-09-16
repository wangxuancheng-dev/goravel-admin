package middleware

import (
	"testing"

	"github.com/goravel/framework/facades"
)

func TestMatchCorsOriginPattern(t *testing.T) {
	patterns := []string{
		"https://*.example.com",
		"http://localhost:3007",
		"*.cdn.test",
	}

	cases := []struct {
		origin string
		want   bool
	}{
		{"https://acme.example.com", true},
		{"https://foo.bar.example.com", true},
		{"http://acme.example.com", false}, // scheme mismatch
		{"https://example.com", true},      // *.suffix also matches apex in matchDomain
		{"http://localhost:3007", false},   // exact (no *) patterns ignored here
		{"https://x.cdn.test", true},
		{"https://evil.com", false},
	}

	for _, tc := range cases {
		if got := matchCorsOriginPattern(tc.origin, patterns); got != tc.want {
			t.Fatalf("matchCorsOriginPattern(%q)=%v want %v", tc.origin, got, tc.want)
		}
	}

	// Exact list entry is not this function's job; host-only wildcard works:
	if !matchCorsOriginPattern("https://a.cdn.test", []string{"*.cdn.test"}) {
		t.Fatal("expected host-only wildcard match")
	}
}

func TestIsCorsBaseDomainHost(t *testing.T) {
	prev := facades.Config().GetString("tenancy.base_domain")
	t.Cleanup(func() { facades.Config().Add("tenancy.base_domain", prev) })

	facades.Config().Add("tenancy.base_domain", "example.com")
	if !isCorsBaseDomainHost("acme.example.com") {
		t.Fatal("tenant subdomain")
	}
	if !isCorsBaseDomainHost("a.b.example.com") {
		t.Fatal("nested subdomain")
	}
	if !isCorsBaseDomainHost("example.com") {
		t.Fatal("apex")
	}
	if isCorsBaseDomainHost("crm.customer.com") {
		t.Fatal("vanity must not match base")
	}

	facades.Config().Add("tenancy.base_domain", "")
	if isCorsBaseDomainHost("acme.example.com") {
		t.Fatal("empty base disables auto subdomain CORS")
	}
}

func TestResolveCorsOriginExactAndWildcard(t *testing.T) {
	ok, acao := resolveCorsOrigin("https://admin.example.com", []string{"https://admin.example.com"})
	if !ok || acao != "https://admin.example.com" {
		t.Fatalf("exact: ok=%v acao=%q", ok, acao)
	}

	ok, acao = resolveCorsOrigin("https://t1.example.com", []string{"https://*.example.com"})
	if !ok || acao != "https://t1.example.com" {
		t.Fatalf("wildcard: ok=%v acao=%q", ok, acao)
	}

	ok, acao = resolveCorsOrigin("https://x.com", []string{"*"})
	if !ok || acao != "*" {
		t.Fatalf("star: ok=%v acao=%q", ok, acao)
	}
}
