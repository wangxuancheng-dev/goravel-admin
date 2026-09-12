package services

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"goravel/app/models"
)

func TestPruneTenantBackups(t *testing.T) {
	dir := t.TempDir()
	names := []string{"20260101_010101.sql", "20260102_010101.sql", "20260103_010101.sql"}
	for _, n := range names {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("--"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	removed, err := pruneTenantBackups(dir, 2)
	if err != nil {
		t.Fatal(err)
	}
	if removed != 1 {
		t.Fatalf("removed=%d want 1", removed)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("left=%d want 2", len(entries))
	}
	if _, err := os.Stat(filepath.Join(dir, "20260101_010101.sql")); !os.IsNotExist(err) {
		t.Fatal("oldest backup should be pruned")
	}
}

func TestPruneTenantBackupsNoopWhenKeepZero(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "a.sql"), []byte("--"), 0o644)
	removed, err := pruneTenantBackups(dir, 0)
	if err != nil || removed != 0 {
		t.Fatalf("removed=%d err=%v", removed, err)
	}
}

func TestTenantOpBusy(t *testing.T) {
	if tenantOpBusy(nil) {
		t.Fatal("nil should not be busy")
	}
	if tenantOpBusy(&models.Tenant{ProvisionStatus: models.TenantProvisionReady}) {
		t.Fatal("ready idle should not be busy")
	}
	now := time.Now()
	if !tenantOpBusy(&models.Tenant{
		ProvisionStatus: models.TenantProvisionMigrating,
		LastOpAt:        &now,
	}) {
		t.Fatal("migrating should be busy")
	}
	if !tenantOpBusy(&models.Tenant{
		LastOpStatus: models.TenantOpStatusQueued,
		LastOpAt:     &now,
	}) {
		t.Fatal("queued should be busy")
	}
	stale := time.Now().Add(-31 * time.Minute)
	if tenantOpBusy(&models.Tenant{
		ProvisionStatus: models.TenantProvisionMigrating,
		LastOpStatus:    models.TenantOpStatusQueued,
		LastOpAt:        &stale,
	}) {
		t.Fatal("stale migrating/queued should allow retry")
	}
	if tenantOpBusy(&models.Tenant{
		LastOpStatus: models.TenantOpStatusQueued,
	}) {
		t.Fatal("queued without last_op_at should allow retry")
	}
}
