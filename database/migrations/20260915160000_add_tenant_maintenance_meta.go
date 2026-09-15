package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
)

type M20260915160000AddTenantMaintenanceMeta struct{}

func (m *M20260915160000AddTenantMaintenanceMeta) Signature() string {
	return "20260915160000_add_tenant_maintenance_meta"
}

func (m *M20260915160000AddTenantMaintenanceMeta) Up() error {
	if SkipOnTenantConnection() {
		return nil
	}
	if !facades.Schema().HasTable("tenants") {
		return nil
	}
	return facades.Schema().Table("tenants", func(table schema.Blueprint) {
		if !facades.Schema().HasColumn("tenants", "maintenance") {
			table.Boolean("maintenance").Default(false).Comment("tenant maintenance mode blocks business traffic")
		}
		if !facades.Schema().HasColumn("tenants", "maintenance_message") {
			table.String("maintenance_message", 500).Nullable().Comment("optional maintenance message for clients")
		}
	})
}

func (m *M20260915160000AddTenantMaintenanceMeta) Down() error {
	if SkipOnTenantConnection() {
		return nil
	}
	if !facades.Schema().HasTable("tenants") {
		return nil
	}
	return facades.Schema().Table("tenants", func(table schema.Blueprint) {
		if facades.Schema().HasColumn("tenants", "maintenance") {
			table.DropColumn("maintenance")
		}
		if facades.Schema().HasColumn("tenants", "maintenance_message") {
			table.DropColumn("maintenance_message")
		}
	})
}
