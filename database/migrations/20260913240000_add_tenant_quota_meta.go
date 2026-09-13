package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
)

type M20260913240000AddTenantQuotaMeta struct{}

func (m *M20260913240000AddTenantQuotaMeta) Signature() string {
	return "20260913240000_add_tenant_quota_meta"
}

func (m *M20260913240000AddTenantQuotaMeta) Up() error {
	if SkipOnTenantConnection() {
		return nil
	}
	if !facades.Schema().HasTable("tenants") {
		return nil
	}
	return facades.Schema().Table("tenants", func(table schema.Blueprint) {
		if !facades.Schema().HasColumn("tenants", "storage_limit_bytes") {
			table.BigInteger("storage_limit_bytes").Default(0).Comment("Object storage quota bytes; 0=unlimited")
		}
		if !facades.Schema().HasColumn("tenants", "traffic_limit_bytes") {
			table.BigInteger("traffic_limit_bytes").Default(0).Comment("Monthly transfer quota bytes; 0=unlimited")
		}
	})
}

func (m *M20260913240000AddTenantQuotaMeta) Down() error {
	if SkipOnTenantConnection() {
		return nil
	}
	if !facades.Schema().HasTable("tenants") {
		return nil
	}
	return facades.Schema().Table("tenants", func(table schema.Blueprint) {
		for _, col := range []string{"storage_limit_bytes", "traffic_limit_bytes"} {
			if facades.Schema().HasColumn("tenants", col) {
				table.DropColumn(col)
			}
		}
	})
}
