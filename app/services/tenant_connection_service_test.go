package services

import "testing"

func TestNormalizeTenantCode(t *testing.T) {
	ok, err := NormalizeTenantCode("Acme_01")
	if err != nil || ok != "acme_01" {
		t.Fatalf("got %q err=%v", ok, err)
	}
	if _, err := NormalizeTenantCode("1bad"); err == nil {
		t.Fatal("expected error for leading digit")
	}
	if _, err := NormalizeTenantCode("Bad-Code"); err == nil {
		t.Fatal("expected error for hyphen")
	}
}

func TestResolveTenantIsolation(t *testing.T) {
	got, err := ResolveTenantIsolation("mysql", "")
	if err != nil || got != "database" {
		t.Fatalf("mysql default: %q %v", got, err)
	}
	if _, err := ResolveTenantIsolation("mysql", "schema"); err == nil {
		t.Fatal("mysql schema should fail")
	}
	got, err = ResolveTenantIsolation("postgres", "schema")
	if err != nil || got != "schema" {
		t.Fatalf("pg schema: %q %v", got, err)
	}
	got, err = ResolveTenantIsolation("postgres", "database")
	if err != nil || got != "database" {
		t.Fatalf("pg database: %q %v", got, err)
	}
}

func TestTenantConnectionName(t *testing.T) {
	if got := TenantConnectionName(12); got != "tenant_12" {
		t.Fatalf("got %q", got)
	}
}
