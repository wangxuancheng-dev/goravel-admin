package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
)

type M20260913223000AddTenantOpLogMeta struct{}

func (r *M20260913223000AddTenantOpLogMeta) Signature() string {
	return "20260913223000_add_tenant_op_log_meta"
}

func (r *M20260913223000AddTenantOpLogMeta) Up() error {
	if SkipOnTenantConnection() {
		return nil
	}
	if !facades.Schema().HasTable("tenant_op_logs") {
		return nil
	}
	return facades.Schema().Table("tenant_op_logs", func(table schema.Blueprint) {
		if !facades.Schema().HasColumn("tenant_op_logs", "batch_id") {
			table.String("batch_id", 64).Nullable().Comment("batch ops correlation id")
			table.Index("batch_id")
		}
		if !facades.Schema().HasColumn("tenant_op_logs", "operator_id") {
			table.UnsignedBigInteger("operator_id").Nullable().Comment("platform_admins.id")
			table.Index("operator_id")
		}
		if !facades.Schema().HasColumn("tenant_op_logs", "operator_name") {
			table.String("operator_name", 100).Nullable().Comment("operator display name")
		}
	})
}

func (r *M20260913223000AddTenantOpLogMeta) Down() error {
	if SkipOnTenantConnection() {
		return nil
	}
	if !facades.Schema().HasTable("tenant_op_logs") {
		return nil
	}
	return facades.Schema().Table("tenant_op_logs", func(table schema.Blueprint) {
		if facades.Schema().HasColumn("tenant_op_logs", "batch_id") {
			table.DropColumn("batch_id")
		}
		if facades.Schema().HasColumn("tenant_op_logs", "operator_id") {
			table.DropColumn("operator_id")
		}
		if facades.Schema().HasColumn("tenant_op_logs", "operator_name") {
			table.DropColumn("operator_name")
		}
	})
}
