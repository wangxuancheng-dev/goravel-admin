package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
)

type M20260916140000AddTenantHealthMeta struct{}

func (r *M20260916140000AddTenantHealthMeta) Signature() string {
	return "20260916140000_add_tenant_health_meta"
}

func (r *M20260916140000AddTenantHealthMeta) Up() error {
	if SkipOnTenantConnection() {
		return nil
	}
	if !facades.Schema().HasTable("tenants") {
		return nil
	}
	return facades.Schema().Table("tenants", func(table schema.Blueprint) {
		if !facades.Schema().HasColumn("tenants", "health_status") {
			table.String("health_status", 32).Default("unknown").Comment("ok|warn|fail|unknown")
		}
		if !facades.Schema().HasColumn("tenants", "health_checked_at") {
			table.Timestamp("health_checked_at").Nullable()
		}
		if !facades.Schema().HasColumn("tenants", "health_issues") {
			table.Text("health_issues").Nullable().Comment("json array of issue codes")
		}
		if !facades.Schema().HasColumn("tenants", "last_ping_ok") {
			table.Boolean("last_ping_ok").Default(false)
		}
		if !facades.Schema().HasColumn("tenants", "last_ping_ms") {
			table.Integer("last_ping_ms").Default(0)
		}
	})
}

func (r *M20260916140000AddTenantHealthMeta) Down() error {
	if SkipOnTenantConnection() {
		return nil
	}
	return facades.Schema().Table("tenants", func(table schema.Blueprint) {
		table.DropColumn("health_status")
		table.DropColumn("health_checked_at")
		table.DropColumn("health_issues")
		table.DropColumn("last_ping_ok")
		table.DropColumn("last_ping_ms")
	})
}