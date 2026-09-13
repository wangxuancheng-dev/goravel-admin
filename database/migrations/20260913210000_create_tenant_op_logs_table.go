package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
)

type M20260913210000CreateTenantOpLogsTable struct{}

func (r *M20260913210000CreateTenantOpLogsTable) Signature() string {
	return "20260913210000_create_tenant_op_logs_table"
}

func (r *M20260913210000CreateTenantOpLogsTable) Up() error {
	if SkipOnTenantConnection() {
		return nil
	}
	if facades.Schema().HasTable("tenant_op_logs") {
		return nil
	}
	return facades.Schema().Create("tenant_op_logs", func(table schema.Blueprint) {
		table.ID()
		table.UnsignedBigInteger("tenant_id").Comment("tenants.id")
		table.String("code", 64).Comment("tenant code snapshot")
		table.String("op", 32).Comment("migrate|seed|backup|restore")
		table.String("status", 32).Comment("queued|running|success|failed")
		table.Text("message").Nullable()
		table.Timestamp("started_at").Nullable()
		table.Timestamp("finished_at").Nullable()
		table.Timestamps()
		table.Index("tenant_id")
		table.Index("code")
		table.Index("op")
		table.Index("status")
	})
}

func (r *M20260913210000CreateTenantOpLogsTable) Down() error {
	if SkipOnTenantConnection() {
		return nil
	}
	return facades.Schema().DropIfExists("tenant_op_logs")
}
