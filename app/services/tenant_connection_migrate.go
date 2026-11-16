package services

import (
	"fmt"
	"strings"
	"time"

	schemaContract "github.com/goravel/framework/contracts/database/schema"
	contractsseeder "github.com/goravel/framework/contracts/database/seeder"
	"github.com/goravel/framework/database/migration"
	"github.com/goravel/framework/facades"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/models"
)

// SeedTenant runs db:seed on the tenant connection.
// seeders empty => all registered seeders; otherwise same as db:seed --seeder=Name.
// Uses per-goroutine Schema+Orm binds when routers are installed (parallel-safe with migrate).
// Marks last_op=seed + last_op_status so CLI seed-all failures are visible in the platform UI
// (same as MigrateTenant). Does not change provision_status (schema may already be ready).
func (s *TenantConnectionService) SeedTenant(tenant *models.Tenant, seeders ...string) error {
	if tenant == nil {
		return apperrors.ErrInvalidArgument.WithMessage("tenant is nil")
	}
	runErr := s.seedTenant(tenant, seeders...)
	s.markSeedOpResult(tenant, runErr)
	return runErr
}

// markSeedOpResult persists last_op fields after SeedTenant (CLI + direct callers).
// Queue RunTenantOp may overwrite these with the enclosing op (e.g. migrate+seed).
func (s *TenantConnectionService) markSeedOpResult(tenant *models.Tenant, runErr error) {
	if tenant == nil || tenant.ID == 0 {
		return
	}
	now := time.Now()
	if runErr != nil {
		msg := runErr.Error()
		if len(msg) > 2000 {
			msg = msg[:2000]
		}
		_, _ = appfacades.PlatformQuery().Model(tenant).Update(map[string]any{
			"last_op":         models.TenantOpSeed,
			"last_op_status":  models.TenantOpStatusFailed,
			"last_op_message": msg,
			"last_op_at":      now,
		})
		tenant.LastOp = models.TenantOpSeed
		tenant.LastOpStatus = models.TenantOpStatusFailed
		tenant.LastOpMessage = msg
		tenant.LastOpAt = &now
		return
	}
	_, _ = appfacades.PlatformQuery().Model(tenant).Update(map[string]any{
		"last_op":         models.TenantOpSeed,
		"last_op_status":  models.TenantOpStatusSuccess,
		"last_op_message": "seed ok",
		"last_op_at":      now,
	})
	tenant.LastOp = models.TenantOpSeed
	tenant.LastOpStatus = models.TenantOpStatusSuccess
	tenant.LastOpMessage = "seed ok"
	tenant.LastOpAt = &now
}

// seedTenant runs the seeders on this goroutine with a fresh tenant Schema bound, so
// seeders' facades.Orm() / facades.Schema() hit the tenant database.
func (s *TenantConnectionService) seedTenant(tenant *models.Tenant, seeders ...string) error {
	sch, err := s.openTenantSchema(tenant)
	if err != nil {
		return err
	}
	return appfacades.WithBoundTenantSchema(sch, func() error {
		// Call seeders on this goroutine. Artisan.Call runs Handle in a NEW goroutine
		// (framework console), which would not see the binding.
		if err := runTenantSeeders(seeders...); err != nil {
			return err
		}
		n, err := sch.Orm().Query().Table("admins").Count()
		if err != nil {
			return fmt.Errorf("seed verify failed: %w", err)
		}
		if n == 0 {
			return fmt.Errorf("seed failed: admins empty on %s", sch.Orm().DatabaseName())
		}
		return nil
	})
}

// runTenantSeeders runs registered seeders on the current goroutine (same as
// db:seed --force [--seeder=...], without Artisan's extra Handle goroutine).
func runTenantSeeders(names ...string) error {
	facade := facades.Seeder()
	if facade == nil {
		return fmt.Errorf("seeder facade unavailable")
	}
	var list []contractsseeder.Seeder
	trimmed := make([]string, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name != "" {
			trimmed = append(trimmed, name)
		}
	}
	if len(trimmed) == 0 {
		list = facade.GetSeeders()
	} else {
		list = make([]contractsseeder.Seeder, 0, len(trimmed))
		for _, name := range trimmed {
			item := facade.GetSeeder(name)
			if item == nil {
				return fmt.Errorf("seeder not found: %s", name)
			}
			list = append(list, item)
		}
	}
	if len(list) == 0 {
		return nil
	}
	return facade.Call(list)
}

// MigrateTenant runs migrations on the tenant connection and marks provision_status=ready.
// Uses a per-goroutine Schema bind (no global database.default flip) so tenant:migrate-all
// concurrency and long-running workers can run true parallel DDL.
func (s *TenantConnectionService) MigrateTenant(tenant *models.Tenant) error {
	var migCount int64
	err := s.migrateTenant(tenant, &migCount)
	now := time.Now()
	if err != nil {
		_ = s.SetProvisionStatus(tenant, models.TenantProvisionFailed)
		msg := err.Error()
		if len(msg) > 2000 {
			msg = msg[:2000]
		}
		_, _ = appfacades.PlatformQuery().Model(tenant).Update(map[string]any{
			"last_migrate_error": msg,
			"last_op":            models.TenantOpMigrate,
			"last_op_status":     models.TenantOpStatusFailed,
			"last_op_message":    msg,
			"last_op_at":         now,
		})
		tenant.LastMigrateError = msg
		tenant.LastOp = models.TenantOpMigrate
		tenant.LastOpStatus = models.TenantOpStatusFailed
		tenant.LastOpMessage = msg
		tenant.LastOpAt = &now
		return err
	}
	if err := s.SetProvisionStatus(tenant, models.TenantProvisionReady); err != nil {
		return err
	}
	_, _ = appfacades.PlatformQuery().Model(tenant).Update(map[string]any{
		"last_migrate_error":     "",
		"migrated_at":            now,
		"schema_migration_count": migCount,
		"last_op":                models.TenantOpMigrate,
		"last_op_status":         models.TenantOpStatusSuccess,
		"last_op_message":        "migrate ok",
		"last_op_at":             now,
	})
	tenant.LastMigrateError = ""
	tenant.MigratedAt = &now
	tenant.SchemaMigrationCount = migCount
	tenant.LastOp = models.TenantOpMigrate
	tenant.LastOpStatus = models.TenantOpStatusSuccess
	tenant.LastOpMessage = "migrate ok"
	tenant.LastOpAt = &now
	return nil
}

// RollbackTenant rolls back migrations on the tenant connection.
// step>0: last N migration files; migBatch>0: that migrations.batch; both 0: last batch (GetLast).
// Uses BindTenantSchema so it can run in parallel with TENANCY_MIGRATE_CONCURRENCY.
func (s *TenantConnectionService) RollbackTenant(tenant *models.Tenant, step, migBatch int) error {
	if step < 0 {
		step = 0
	}
	if migBatch < 0 {
		migBatch = 0
	}
	var migCount int64
	err := s.rollbackTenant(tenant, step, migBatch, &migCount)
	now := time.Now()
	if err != nil {
		msg := err.Error()
		if len(msg) > 2000 {
			msg = msg[:2000]
		}
		_, _ = appfacades.PlatformQuery().Model(tenant).Update(map[string]any{
			"last_op":         models.TenantOpRollback,
			"last_op_status":  models.TenantOpStatusFailed,
			"last_op_message": msg,
			"last_op_at":      now,
		})
		tenant.LastOp = models.TenantOpRollback
		tenant.LastOpStatus = models.TenantOpStatusFailed
		tenant.LastOpMessage = msg
		tenant.LastOpAt = &now
		return err
	}

	provision := models.TenantProvisionReady
	// Bind briefly to check admins still exists after rollback.
	if checkErr := s.withTenantHasAdmins(tenant); checkErr != nil {
		provision = models.TenantProvisionPending
	}
	_ = s.SetProvisionStatus(tenant, provision)
	msg := fmt.Sprintf("rollback ok (step=%d batch=%d)", step, migBatch)
	_, _ = appfacades.PlatformQuery().Model(tenant).Update(map[string]any{
		"schema_migration_count": migCount,
		"last_op":                models.TenantOpRollback,
		"last_op_status":         models.TenantOpStatusSuccess,
		"last_op_message":        msg,
		"last_op_at":             now,
		"last_migrate_error":     "",
	})
	tenant.SchemaMigrationCount = migCount
	tenant.LastOp = models.TenantOpRollback
	tenant.LastOpStatus = models.TenantOpStatusSuccess
	tenant.LastOpMessage = msg
	tenant.LastOpAt = &now
	tenant.LastMigrateError = ""
	return nil
}

func (s *TenantConnectionService) withTenantHasAdmins(tenant *models.Tenant) error {
	sch, err := s.openTenantSchema(tenant)
	if err != nil {
		return err
	}
	if !sch.HasTable("admins") {
		return fmt.Errorf("admins missing")
	}
	return nil
}

// rollbackTenant rolls back on a fresh tenant Schema bound to this goroutine (so
// migrations' Down() via facades.Schema() hit the tenant); safe to run in parallel.
func (s *TenantConnectionService) rollbackTenant(tenant *models.Tenant, step, migBatch int, migCount *int64) error {
	sch, err := s.openTenantSchema(tenant)
	if err != nil {
		return err
	}
	return appfacades.WithBoundTenantSchema(sch, func() error {
		migrator := migration.NewMigrator(facades.Artisan(), sch, "migrations")
		if err := migrator.Rollback(step, migBatch); err != nil {
			return err
		}
		return countTenantMigrations(sch, migCount)
	})
}

// migrateTenant migrates on a fresh tenant Schema bound to this goroutine (so
// migrations' Up() via facades.Schema() hit the tenant); safe to run in parallel.
func (s *TenantConnectionService) migrateTenant(tenant *models.Tenant, migCount *int64) error {
	sch, err := s.openTenantSchema(tenant)
	if err != nil {
		return err
	}
	return appfacades.WithBoundTenantSchema(sch, func() error {
		// Call Migrator.Run directly: Artisan migrate returns nil even on failure.
		migrator := migration.NewMigrator(facades.Artisan(), sch, "migrations")
		if err := migrator.Run(); err != nil {
			return err
		}
		if !sch.HasTable("admins") {
			return fmt.Errorf("migrate failed: admins table missing on %s", sch.Orm().DatabaseName())
		}
		return countTenantMigrations(sch, migCount)
	})
}

func countTenantMigrations(sch schemaContract.Schema, migCount *int64) error {
	if !sch.HasTable("migrations") {
		return nil
	}
	n, err := sch.Orm().Query().Table("migrations").Count()
	if err != nil {
		return err
	}
	*migCount = n
	return nil
}

// openTenantSchema registers the tenant connection and returns a fresh Schema on it.
// Each caller gets its own instance: the migrator mutates the Schema it runs on.
func (s *TenantConnectionService) openTenantSchema(tenant *models.Tenant) (schemaContract.Schema, error) {
	c, err := s.ensureConn(tenant)
	if err != nil {
		return nil, err
	}
	sch, err := c.NewSchema()
	if err != nil {
		return nil, apperrors.ErrTenantConnectionFailed.WithError(err)
	}
	return sch, nil
}
