package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
)

type M20260913250000DropTenantTrafficLimit struct{}

func (m *M20260913250000DropTenantTrafficLimit) Signature() string {
	return "20260913250000_drop_tenant_traffic_limit"
}

func (m *M20260913250000DropTenantTrafficLimit) Up() error {
	if SkipOnTenantConnection() {
		return nil
	}
	if !facades.Schema().HasTable("tenants") {
		return nil
	}
	if !facades.Schema().HasColumn("tenants", "traffic_limit_bytes") {
		return nil
	}
	return facades.Schema().Table("tenants", func(table schema.Blueprint) {
		table.DropColumn("traffic_limit_bytes")
	})
}

func (m *M20260913250000DropTenantTrafficLimit) Down() error {
	if SkipOnTenantConnection() {
		return nil
	}
	if !facades.Schema().HasTable("tenants") {
		return nil
	}
	if facades.Schema().HasColumn("tenants", "traffic_limit_bytes") {
		return nil
	}
	return facades.Schema().Table("tenants", func(table schema.Blueprint) {
		table.BigInteger("traffic_limit_bytes").Default(0)
	})
}
