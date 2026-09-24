package tenancy

import (
	"github.com/goravel/framework/facades"
)

const (
	// DefaultBackupConcurrency parallel dumps inside one tenant_ops_fleet backup job.
	DefaultBackupConcurrency = 2
	// MaxBackupConcurrency caps env override (dumps are heavier than migrate DDL).
	MaxBackupConcurrency = 50
)

// ClampBackupConcurrency normalizes fleet backup parallelism.
func ClampBackupConcurrency(n int) int {
	if n < 1 {
		return DefaultBackupConcurrency
	}
	if n > MaxBackupConcurrency {
		return MaxBackupConcurrency
	}
	return n
}

// BackupConcurrency returns TENANCY_BACKUP_CONCURRENCY (clamped).
func BackupConcurrency() int {
	return ClampBackupConcurrency(facades.Config().GetInt("tenancy.backup_concurrency", DefaultBackupConcurrency))
}
