package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
)

type M20260917220000CreatePlatformLoginLogsTable struct{}

func (r *M20260917220000CreatePlatformLoginLogsTable) Signature() string {
	return "20260917220000_create_platform_login_logs_table"
}

func (r *M20260917220000CreatePlatformLoginLogsTable) Up() error {
	if SkipOnTenantConnection() {
		return nil
	}
	if facades.Schema().HasTable("platform_login_logs") {
		return nil
	}
	return facades.Schema().Create("platform_login_logs", func(table schema.Blueprint) {
		table.ID()
		table.UnsignedBigInteger("admin_id").Nullable().Comment("platform_admins.id")
		table.String("username", 50).Nullable().Comment("username")
		table.String("ip", 50).Nullable().Comment("ip")
		table.String("user_agent", 500).Nullable().Comment("user agent")
		table.String("location", 100).Nullable().Comment("geo location")
		table.UnsignedTinyInteger("status").Default(0).Comment("1 success 0 failed")
		table.String("message", 255).Nullable().Comment("result message key")
		table.Text("request").Nullable().Comment("sanitized request body")
		table.Timestamps()
		table.Index("admin_id")
		table.Index("username")
		table.Index("ip")
		table.Index("status")
	})
}

func (r *M20260917220000CreatePlatformLoginLogsTable) Down() error {
	if SkipOnTenantConnection() {
		return nil
	}
	return facades.Schema().DropIfExists("platform_login_logs")
}
