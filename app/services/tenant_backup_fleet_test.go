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

func TestResolveBackupScheduleMode(t *testing.T) {
	prev := facades.Config().GetString("tenancy.backup_schedule_mode", "full")
	t.Cleanup(func() {
		facades.Config().Add("tenancy.backup_schedule_mode", prev)
	})
	facades.Config().Add("tenancy.backup_schedule_mode", "rotate")
	if got := ResolveBackupScheduleMode(); got != BackupScheduleModeRotate {
		t.Fatalf("got %s", got)
	}
	facades.Config().Add("tenancy.backup_schedule_mode", "FULL")
	if got := ResolveBackupScheduleMode(); got != BackupScheduleModeFull {
		t.Fatalf("got %s", got)
	}
	facades.Config().Add("tenancy.backup_schedule_mode", "weird")
	if got := ResolveBackupScheduleMode(); got != BackupScheduleModeFull {
		t.Fatalf("default full got %s", got)
	}
}
