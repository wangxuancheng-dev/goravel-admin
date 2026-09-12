package commands

import (
	"strings"
	"testing"

	"goravel/app/models"
)

func TestPostgresRestoreArgsSetsSearchPathForSchemaIsolation(t *testing.T) {
	tenant := &models.Tenant{
		Driver:    models.TenantDriverPostgres,
		Isolation: models.TenantIsolationSchema,
		Schema:    "tenant_acme",
	}
	args, env := postgresRestoreArgs("127.0.0.1", 5432, "u", "p", "db", "/tmp/a.sql", tenant)
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "-f /tmp/a.sql") {
		t.Fatalf("args missing file: %v", args)
	}
	if !strings.Contains(joined, "ON_ERROR_STOP=1") {
		t.Fatalf("args missing ON_ERROR_STOP: %v", args)
	}
	found := false
	for _, e := range env {
		if e == "PGOPTIONS=--search_path=tenant_acme" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected search_path env, got %v", env)
	}
}

func TestPostgresRestoreArgsSkipsSearchPathForDatabaseIsolation(t *testing.T) {
	tenant := &models.Tenant{
		Driver:    models.TenantDriverPostgres,
		Isolation: models.TenantIsolationDatabase,
		Schema:    "ignored",
	}
	_, env := postgresRestoreArgs("127.0.0.1", 5432, "u", "p", "db", "/tmp/a.sql", tenant)
	for _, e := range env {
		if strings.HasPrefix(e, "PGOPTIONS=") {
			t.Fatalf("unexpected PGOPTIONS for database isolation: %s", e)
		}
	}
}
