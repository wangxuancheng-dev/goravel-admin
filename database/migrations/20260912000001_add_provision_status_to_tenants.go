package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
)

type M20260912000001AddProvisionStatusToTenants struct{}

func (m *M20260912000001AddProvisionStatusToTenants) Signature() string {
	return "20260912000001_add_provision_status_to_tenants"
}

func (m *M20260912000001AddProvisionStatusToTenants) Up() error {
	if !facades.Schema().HasTable("tenants") {
		return nil
	}
	if facades.Schema().HasColumn("tenants", "provision_status") {
		return nil
	}
	if err := facades.Schema().Table("tenants", func(table schema.Blueprint) {
		table.String("provision_status", 32).Default("ready").Comment("pending|ready|failed")
		table.Index("provision_status")
	}); err != nil {
		return err
	}
	// Existing rows become ready so current tenants keep accepting traffic.
	_, err := facades.Orm().Query().Exec("UPDATE tenants SET provision_status = ? WHERE provision_status IS NULL OR provision_status = ''", "ready")
	return err
}

func (m *M20260912000001AddProvisionStatusToTenants) Down() error {
	if !facades.Schema().HasTable("tenants") {
		return nil
	}
	if !facades.Schema().HasColumn("tenants", "provision_status") {
		return nil
	}
	return facades.Schema().Table("tenants", func(table schema.Blueprint) {
		table.DropColumn("provision_status")
	})
}
