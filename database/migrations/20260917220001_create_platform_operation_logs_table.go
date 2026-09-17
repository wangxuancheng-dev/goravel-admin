package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
)

type M20260917220001CreatePlatformOperationLogsTable struct{}

func (r *M20260917220001CreatePlatformOperationLogsTable) Signature() string {
	return "20260917220001_create_platform_operation_logs_table"
}

func (r *M20260917220001CreatePlatformOperationLogsTable) Up() error {
	if SkipOnTenantConnection() {
		return nil
	}
	if facades.Schema().HasTable("platform_operation_logs") {
		return nil
	}
	return facades.Schema().Create("platform_operation_logs", func(table schema.Blueprint) {
		table.ID()
		table.UnsignedBigInteger("admin_id").Nullable().Comment("platform_admins.id")
		table.String("username", 50).Nullable().Comment("username snapshot")
		table.String("method", 10).Nullable().Comment("http method")
		table.String("path", 255).Nullable().Comment("request path")
		table.String("title", 255).Nullable().Comment("operation title")
		table.String("ip", 50).Nullable().Comment("ip")
		table.String("user_agent", 500).Nullable().Comment("user agent")
		table.Text("request").Nullable().Comment("sanitized request body")
		table.UnsignedTinyInteger("status").Default(1).Comment("1 success 0 failed")
		table.Text("error_msg").Nullable().Comment("error message")
		table.Integer("duration").Nullable().Comment("duration ms")
		table.Timestamps()
		table.Index("admin_id")
		table.Index("username")
		table.Index("method")
		table.Index("path")
		table.Index("status")
	})
}

func (r *M20260917220001CreatePlatformOperationLogsTable) Down() error {
	if SkipOnTenantConnection() {
		return nil
	}
	return facades.Schema().DropIfExists("platform_operation_logs")
}
