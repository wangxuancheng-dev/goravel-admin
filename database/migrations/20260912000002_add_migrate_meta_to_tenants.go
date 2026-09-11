package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
)

type M20260912000002AddMigrateMetaToTenants struct{}

func (m *M20260912000002AddMigrateMetaToTenants) Signature() string {
	return "20260912000002_add_migrate_meta_to_tenants"
}

func (m *M20260912000002AddMigrateMetaToTenants) Up() error {
	if SkipOnTenantConnection() {
		return nil
	}
	if !facades.Schema().HasTable("tenants") {
		return nil
	}
	return facades.Schema().Table("tenants", func(table schema.Blueprint) {
		if !facades.Schema().HasColumn("tenants", "last_migrate_error") {
			table.Text("last_migrate_error").Nullable().Comment("最近一次 migrate 错误")
		}
		if !facades.Schema().HasColumn("tenants", "migrated_at") {
			table.Timestamp("migrated_at").Nullable().Comment("最近一次 migrate 成功时间")
		}
	})
}

func (m *M20260912000002AddMigrateMetaToTenants) Down() error {
	if SkipOnTenantConnection() {
		return nil
	}
	if !facades.Schema().HasTable("tenants") {
		return nil
	}
	return facades.Schema().Table("tenants", func(table schema.Blueprint) {
		if facades.Schema().HasColumn("tenants", "last_migrate_error") {
			table.DropColumn("last_migrate_error")
		}
		if facades.Schema().HasColumn("tenants", "migrated_at") {
			table.DropColumn("migrated_at")
		}
	})
}
