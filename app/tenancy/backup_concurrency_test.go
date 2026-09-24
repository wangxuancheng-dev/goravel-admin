package tenancy

import "testing"

func TestClampBackupConcurrency(t *testing.T) {
	if got := ClampBackupConcurrency(0); got != DefaultBackupConcurrency {
		t.Fatalf("zero: %d", got)
	}
	if got := ClampBackupConcurrency(4); got != 4 {
		t.Fatalf("4: %d", got)
	}
	if got := ClampBackupConcurrency(999); got != MaxBackupConcurrency {
		t.Fatalf("cap: %d", got)
	}
}
