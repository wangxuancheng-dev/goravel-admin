package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
)

type M20260915140000AddTenantSchemaMigrationCount struct{}

func (m *M20260915140000AddTenantSchemaMigrationCount) Signature() string {
	return "20260915140000_add_tenant_schema_migration_count"
}

func (m *M20260915140000AddTenantSchemaMigrationCount) Up() error {
	if SkipOnTenantConnection() {
		return nil
	}
	if !facades.Schema().HasTable("tenants") {
		return nil
	}
	return facades.Schema().Table("tenants", func(table schema.Blueprint) {
		if !facades.Schema().HasColumn("tenants", "schema_migration_count") {
			table.Integer("schema_migration_count").Default(0).Comment("tenant migrations table row count after last successful migrate")
		}
	})
}

func (m *M20260915140000AddTenantSchemaMigrationCount) Down() error {
	if SkipOnTenantConnection() {
		return nil
	}
	if !facades.Schema().HasTable("tenants") {
		return nil
	}
	return facades.Schema().Table("tenants", func(table schema.Blueprint) {
		if facades.Schema().HasColumn("tenants", "schema_migration_count") {
			table.DropColumn("schema_migration_count")
		}
	})
}
