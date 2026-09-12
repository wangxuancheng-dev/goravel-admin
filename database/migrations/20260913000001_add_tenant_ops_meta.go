package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
)

type M20260913000001AddTenantOpsMeta struct{}

func (m *M20260913000001AddTenantOpsMeta) Signature() string {
	return "20260913000001_add_tenant_ops_meta"
}

func (m *M20260913000001AddTenantOpsMeta) Up() error {
	if SkipOnTenantConnection() {
		return nil
	}
	if !facades.Schema().HasTable("tenants") {
		return nil
	}
	return facades.Schema().Table("tenants", func(table schema.Blueprint) {
		if !facades.Schema().HasColumn("tenants", "last_op") {
			table.String("last_op", 32).Nullable().Comment("最近平台运维操作 migrate|seed|backup")
		}
		if !facades.Schema().HasColumn("tenants", "last_op_status") {
			table.String("last_op_status", 32).Nullable().Comment("idle|queued|running|success|failed")
		}
		if !facades.Schema().HasColumn("tenants", "last_op_message") {
			table.Text("last_op_message").Nullable().Comment("最近运维操作结果或错误")
		}
		if !facades.Schema().HasColumn("tenants", "last_op_at") {
			table.Timestamp("last_op_at").Nullable().Comment("最近运维操作时间")
		}
		if !facades.Schema().HasColumn("tenants", "last_backup_path") {
			table.String("last_backup_path", 512).Nullable().Comment("最近一次备份文件路径")
		}
	})
}

func (m *M20260913000001AddTenantOpsMeta) Down() error {
	if SkipOnTenantConnection() {
		return nil
	}
	if !facades.Schema().HasTable("tenants") {
		return nil
	}
	return facades.Schema().Table("tenants", func(table schema.Blueprint) {
		for _, col := range []string{"last_op", "last_op_status", "last_op_message", "last_op_at", "last_backup_path"} {
			if facades.Schema().HasColumn("tenants", col) {
				table.DropColumn(col)
			}
		}
	})
}
