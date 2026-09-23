package tenancy

import (
	"github.com/goravel/framework/facades"
)

const (
	// DefaultMigrateConcurrency is the stock limit for tenant:migrate-all.
	DefaultMigrateConcurrency = 2
	// MaxMigrateConcurrency caps env / --concurrency to avoid accidental DB stampede.
	MaxMigrateConcurrency = 100
)

// ClampMigrateConcurrency normalizes a concurrency value for fleet migrate.
// Values below 1 fall back to DefaultMigrateConcurrency; values above Max are capped.
func ClampMigrateConcurrency(n int) int {
	if n < 1 {
		return DefaultMigrateConcurrency
	}
	if n > MaxMigrateConcurrency {
		return MaxMigrateConcurrency
	}
	return n
}

// MigrateConcurrency returns TENANCY_MIGRATE_CONCURRENCY (clamped).
func MigrateConcurrency() int {
	return ClampMigrateConcurrency(facades.Config().GetInt("tenancy.migrate_concurrency", DefaultMigrateConcurrency))
}
