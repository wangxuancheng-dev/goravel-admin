package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
)

type M20261011000000CreatePlatformSettingsTable struct{}

func (m *M20261011000000CreatePlatformSettingsTable) Signature() string {
	return "20261011000000_create_platform_settings_table"
}

// Up creates the landlord key/value settings table (platform console settings such as captcha type).
func (m *M20261011000000CreatePlatformSettingsTable) Up() error {
	if SkipOnTenantConnection() {
		return nil
	}
	if facades.Schema().HasTable("platform_settings") {
		return nil
	}
	return facades.Schema().Create("platform_settings", func(table schema.Blueprint) {
		table.ID()
		table.String("key", 64).Comment("setting key")
		table.Text("value").Nullable().Comment("setting value")
		table.Timestamps()
		table.Unique("key")
	})
}

func (m *M20261011000000CreatePlatformSettingsTable) Down() error {
	if SkipOnTenantConnection() {
		return nil
	}
	return facades.Schema().DropIfExists("platform_settings")
}
