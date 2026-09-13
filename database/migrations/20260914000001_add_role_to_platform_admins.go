package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
)

type M20260914000001AddRoleToPlatformAdmins struct{}

func (m *M20260914000001AddRoleToPlatformAdmins) Signature() string {
	return "20260914000001_add_role_to_platform_admins"
}

func (m *M20260914000001AddRoleToPlatformAdmins) Up() error {
	if SkipOnTenantConnection() {
		return nil
	}
	if !facades.Schema().HasTable("platform_admins") {
		return nil
	}
	if facades.Schema().HasColumn("platform_admins", "role") {
		return nil
	}
	return facades.Schema().Table("platform_admins", func(table schema.Blueprint) {
		table.String("role", 32).Default("owner").Comment("owner|viewer")
	})
}

func (m *M20260914000001AddRoleToPlatformAdmins) Down() error {
	if SkipOnTenantConnection() {
		return nil
	}
	if !facades.Schema().HasTable("platform_admins") {
		return nil
	}
	if !facades.Schema().HasColumn("platform_admins", "role") {
		return nil
	}
	return facades.Schema().Table("platform_admins", func(table schema.Blueprint) {
		table.DropColumn("role")
	})
}
