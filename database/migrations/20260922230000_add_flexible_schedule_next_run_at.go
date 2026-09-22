package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
)

// M20260922230000AddFlexibleScheduleNextRunAt adds next_run_at for due-row ticks at scale.
type M20260922230000AddFlexibleScheduleNextRunAt struct{}

func (m *M20260922230000AddFlexibleScheduleNextRunAt) Signature() string {
	return "20260922230000_add_flexible_schedule_next_run_at"
}

func (m *M20260922230000AddFlexibleScheduleNextRunAt) Up() error {
	if SkipOnTenantConnection() {
		return nil
	}
	if !facades.Schema().HasTable("flexible_schedules") {
		return nil
	}
	if facades.Schema().HasColumn("flexible_schedules", "next_run_at") {
		return nil
	}
	return facades.Schema().Table("flexible_schedules", func(table schema.Blueprint) {
		table.Timestamp("next_run_at").Nullable()
		table.Index("next_run_at")
	})
}

func (m *M20260922230000AddFlexibleScheduleNextRunAt) Down() error {
	if SkipOnTenantConnection() {
		return nil
	}
	if !facades.Schema().HasTable("flexible_schedules") {
		return nil
	}
	if !facades.Schema().HasColumn("flexible_schedules", "next_run_at") {
		return nil
	}
	return facades.Schema().Table("flexible_schedules", func(table schema.Blueprint) {
		table.DropIndex("next_run_at")
		table.DropColumn("next_run_at")
	})
}
