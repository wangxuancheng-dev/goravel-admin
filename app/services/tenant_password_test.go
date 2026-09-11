package services

import (
	"testing"

	"goravel/app/models"
)

func TestTenantHasPassword(t *testing.T) {
	if TenantHasPassword("") {
		t.Fatal("empty should be false")
	}
	if !TenantHasPassword("plain") {
		t.Fatal("plain should be true")
	}
	if !TenantHasPassword(tenantSecretPrefix + "abc") {
		t.Fatal("sealed should be true")
	}
}

func TestSealTenantPasswordIdempotentPrefix(t *testing.T) {
	already := tenantSecretPrefix + "payload"
	got, err := SealTenantPassword(already)
	if err != nil {
		t.Fatal(err)
	}
	if got != already {
		t.Fatalf("expected unchanged sealed value, got %q", got)
	}
	empty, err := SealTenantPassword("")
	if err != nil || empty != "" {
		t.Fatalf("empty seal: %q %v", empty, err)
	}
}

func TestRevealTenantPasswordLegacyPlain(t *testing.T) {
	got, err := RevealTenantPassword("legacy-secret")
	if err != nil {
		t.Fatal(err)
	}
	if got != "legacy-secret" {
		t.Fatalf("got %q", got)
	}
}

func TestNormalizeTenantDriver(t *testing.T) {
	cases := map[string]string{
		"":           "",
		"MySQL":      "mysql",
		" postgres ": "postgres",
		"pgsql":      "postgres",
		"PostgreSQL": "postgres",
		"mysql":      "mysql",
	}
	for in, want := range cases {
		if got := NormalizeTenantDriver(in); got != want {
			t.Fatalf("%q => %q, want %q", in, got, want)
		}
	}
}

func TestTenantUsesCustomHost(t *testing.T) {
	if TenantUsesCustomHost(nil) {
		t.Fatal("nil")
	}
	if TenantUsesCustomHost(&models.Tenant{Host: ""}) {
		t.Fatal("empty host")
	}
	if TenantUsesCustomHost(&models.Tenant{Host: "  "}) {
		t.Fatal("blank host")
	}
	if !TenantUsesCustomHost(&models.Tenant{Host: "10.0.0.1"}) {
		t.Fatal("custom host")
	}
}
