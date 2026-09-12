package services

import (
	"testing"

	"github.com/goravel/framework/facades"

	appfacades "goravel/app/facades"
	"goravel/app/models"
)

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

func TestValidateTenantCredentials(t *testing.T) {
	prev := facades.Config().GetBool("tenancy.allow_platform_db_credentials", true)
	t.Cleanup(func() {
		facades.Config().Add("tenancy.allow_platform_db_credentials", prev)
	})

	facades.Config().Add("tenancy.allow_platform_db_credentials", true)
	if err := ValidateTenantCredentials("", "", ""); err != nil {
		t.Fatalf("same-host shared creds should allow empty: %v", err)
	}

	facades.Config().Add("tenancy.allow_platform_db_credentials", false)
	if err := ValidateTenantCredentials("", "", ""); err == nil {
		t.Fatal("expected error when shared creds disabled")
	}

	if err := ValidateTenantCredentials("10.0.0.8", "", "secret"); err == nil {
		t.Fatal("remote host requires username")
	}
	if err := ValidateTenantCredentials("10.0.0.8", "u", ""); err == nil {
		t.Fatal("remote host requires password")
	}
	if err := ValidateTenantCredentials("10.0.0.8", "u", "secret"); err != nil {
		t.Fatalf("remote dedicated creds should pass: %v", err)
	}
}

func TestTenantIsProvisionReady(t *testing.T) {
	cases := []struct {
		status string
		want   bool
	}{
		{"", true},
		{models.TenantProvisionReady, true},
		{models.TenantProvisionPending, false},
		{models.TenantProvisionMigrating, false},
		{models.TenantProvisionFailed, false},
	}
	for _, tc := range cases {
		tenant := &models.Tenant{ProvisionStatus: tc.status}
		if got := tenant.IsProvisionReady(); got != tc.want {
			t.Fatalf("status=%q got %v want %v", tc.status, got, tc.want)
		}
	}
}

func TestPlatformOrmQueryIgnoresDefaultFlip(t *testing.T) {
	appfacades.ResetPlatformConnectionNameForTest()
	pinned := appfacades.PlatformConnectionName()
	if pinned == "" || (len(pinned) >= 7 && pinned[:7] == "tenant_") {
		t.Fatalf("unexpected pinned platform connection: %q", pinned)
	}

	prev := facades.Config().GetString("database.default")
	facades.Config().Add("database.default", "tenant_999")
	t.Cleanup(func() {
		facades.Config().Add("database.default", prev)
		appfacades.ResetPlatformConnectionNameForTest()
	})

	if got := appfacades.PlatformConnectionName(); got != pinned {
		t.Fatalf("pin must not follow flipped default: got %q want %q", got, pinned)
	}
}

func TestTenantPostgresSSLMode(t *testing.T) {
	prevTenancy := facades.Config().GetString("tenancy.postgres_sslmode")
	prevDB := facades.Config().GetString("database.connections.postgres.sslmode")
	t.Cleanup(func() {
		facades.Config().Add("tenancy.postgres_sslmode", prevTenancy)
		facades.Config().Add("database.connections.postgres.sslmode", prevDB)
	})

	facades.Config().Add("tenancy.postgres_sslmode", "require")
	if got := tenantPostgresSSLMode(); got != "require" {
		t.Fatalf("tenancy override: got %q", got)
	}

	facades.Config().Add("tenancy.postgres_sslmode", "")
	facades.Config().Add("database.connections.postgres.sslmode", "verify-full")
	if got := tenantPostgresSSLMode(); got != "verify-full" {
		t.Fatalf("db fallback: got %q", got)
	}

	facades.Config().Add("database.connections.postgres.sslmode", "")
	if got := tenantPostgresSSLMode(); got != "disable" {
		t.Fatalf("default: got %q", got)
	}
}

func TestFilterReadyTenants(t *testing.T) {
	in := []models.Tenant{
		{Code: "a", ProvisionStatus: models.TenantProvisionReady},
		{Code: "b", ProvisionStatus: models.TenantProvisionPending},
		{Code: "c", ProvisionStatus: models.TenantProvisionFailed},
		{Code: "d", ProvisionStatus: ""},
	}
	out := filterReadyTenants(in)
	if len(out) != 2 || out[0].Code != "a" || out[1].Code != "d" {
		t.Fatalf("got %+v", out)
	}
}
