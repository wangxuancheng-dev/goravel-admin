package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
)

type M20261010190000AddPlatformFeatureAlwaysOn struct{}

func (m *M20261010190000AddPlatformFeatureAlwaysOn) Signature() string {
	return "20261010190000_add_platform_feature_always_on"
}

func (m *M20261010190000AddPlatformFeatureAlwaysOn) Up() error {
	if SkipOnTenantConnection() {
		return nil
	}
	if !facades.Schema().HasTable("platform_features") {
		return nil
	}
	if facades.Schema().HasColumn("platform_features", "always_on") {
		return nil
	}
	return facades.Schema().Table("platform_features", func(table schema.Blueprint) {
		table.Boolean("always_on").Default(false)
	})
}

func (m *M20261010190000AddPlatformFeatureAlwaysOn) Down() error {
	if SkipOnTenantConnection() {
		return nil
	}
	if !facades.Schema().HasTable("platform_features") {
		return nil
	}
	if !facades.Schema().HasColumn("platform_features", "always_on") {
		return nil
	}
	return facades.Schema().Table("platform_features", func(table schema.Blueprint) {
		table.DropColumn("always_on")
	})
}
