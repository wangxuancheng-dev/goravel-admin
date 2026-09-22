package services

import (
	"testing"

	"github.com/goravel/framework/facades"
)

func TestResolveBackupScheduleBatch(t *testing.T) {
	prev := facades.Config().GetInt("tenancy.backup_schedule_batch", 100)
	t.Cleanup(func() {
		facades.Config().Add("tenancy.backup_schedule_batch", prev)
	})
	facades.Config().Add("tenancy.backup_schedule_batch", 100)
	if got := ResolveBackupScheduleBatch(); got != 100 {
		t.Fatalf("got %d", got)
	}
	facades.Config().Add("tenancy.backup_schedule_batch", 0)
	if got := ResolveBackupScheduleBatch(); got != 100 {
		t.Fatalf("zero fallback got %d", got)
	}
	facades.Config().Add("tenancy.backup_schedule_batch", 9999)
	if got := ResolveBackupScheduleBatch(); got != 500 {
		t.Fatalf("cap got %d", got)
	}
}
