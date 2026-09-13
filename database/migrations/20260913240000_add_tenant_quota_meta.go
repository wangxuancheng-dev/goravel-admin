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
	if err := facades.Schema().Table("tenants", func(table schema.Blueprint) {
		if !facades.Schema().HasColumn("tenants", "storage_limit_bytes") {
			table.BigInteger("storage_limit_bytes").Default(0).Comment("Object storage quota bytes; 0=unlimited")
		}
	}); err != nil {
		return err
	}
	// Drop traffic quota if an earlier draft of this migration added it.
	if facades.Schema().HasColumn("tenants", "traffic_limit_bytes") {
		return facades.Schema().Table("tenants", func(table schema.Blueprint) {
			table.DropColumn("traffic_limit_bytes")
		})
	}
	return nil
}

func (m *M20260913240000AddTenantQuotaMeta) Down() error {
	if SkipOnTenantConnection() {
		return nil
	}
	if !facades.Schema().HasTable("tenants") {
		return nil
	}
	return facades.Schema().Table("tenants", func(table schema.Blueprint) {
		if facades.Schema().HasColumn("tenants", "storage_limit_bytes") {
			table.DropColumn("storage_limit_bytes")
		}
	})
}
