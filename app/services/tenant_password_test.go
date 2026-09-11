package services

import (
	"testing"

	"goravel/app/models"
)

func TestTenantHasPassword(t *testing.T) {
	if TenantHasPassword("") {
		t.Fatal("empty should be false")
	}
	if !TenantHasPassword("x") {
		t.Fatal("non-empty should be true")
	}
}

func TestSealTenantPasswordRejectsSealedInput(t *testing.T) {
	_, err := SealTenantPassword(tenantSecretPrefix + "payload")
	if err == nil {
		t.Fatal("expected error for already-sealed input")
	}
	empty, err := SealTenantPassword("")
	if err != nil || empty != "" {
		t.Fatalf("empty seal: %q %v", empty, err)
	}
}

func TestRevealTenantPasswordRejectsPlaintext(t *testing.T) {
	_, err := RevealTenantPassword("legacy-secret")
	if err == nil {
		t.Fatal("expected error for plaintext")
	}
	got, err := RevealTenantPassword("")
	if err != nil || got != "" {
		t.Fatalf("empty reveal: %q %v", got, err)
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
