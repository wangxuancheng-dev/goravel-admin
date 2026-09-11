package commands

import (
	"os"
	"path/filepath"
	"testing"
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
