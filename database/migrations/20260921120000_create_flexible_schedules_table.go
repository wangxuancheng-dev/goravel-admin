package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
)

type M20260921120000CreateFlexibleSchedulesTable struct{}

func (m *M20260921120000CreateFlexibleSchedulesTable) Signature() string {
	return "20260921120000_create_flexible_schedules_table"
}

func (m *M20260921120000CreateFlexibleSchedulesTable) Up() error {
	if SkipOnTenantConnection() {
		return nil
	}
	if facades.Schema().HasTable("flexible_schedules") {
		return nil
	}
	return facades.Schema().Create("flexible_schedules", func(table schema.Blueprint) {
		table.ID()
		table.String("name", 100)
		table.String("handler", 64)
		table.String("cron_expr", 64)
		table.String("timezone", 64).Default("UTC")
		table.UnsignedBigInteger("tenant_id").Default(0)
		table.Text("payload").Nullable().Comment("handler options JSON object")
		table.Boolean("enabled").Default(true)
		table.Timestamp("last_run_at").Nullable()
		table.String("last_status", 16).Default("never")
		table.Text("last_error").Nullable()
		table.Text("last_output").Nullable()
		table.BigInteger("last_duration_ms").Default(0)
		table.String("last_slot", 32).Default("")
		table.Timestamps()
		table.Index("handler")
		table.Index("tenant_id")
		table.Index("enabled")
	})
}

func (m *M20260921120000CreateFlexibleSchedulesTable) Down() error {
	if SkipOnTenantConnection() {
		return nil
	}
	return facades.Schema().DropIfExists("flexible_schedules")
}
