package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
)

type M20260920150000CreateDemoActivitiesTable struct{}

func (r *M20260920150000CreateDemoActivitiesTable) Signature() string {
	return "20260920150000_create_demo_activities_table"
}

func (r *M20260920150000CreateDemoActivitiesTable) Up() error {
	if facades.Schema().HasTable("demo_activities") {
		return nil
	}
	return facades.Schema().Create("demo_activities", func(table schema.Blueprint) {
		table.BigIncrements("id")
		table.String("title", 120).Comment("demo activity title")
		table.String("schedule_type", 16).Default("once").Comment("once|daily|weekly|monthly|yearly")
		table.Timestamp("start_at").Nullable().Comment("once: absolute start (UTC)")
		table.Timestamp("end_at").Nullable().Comment("once: absolute end (UTC)")
		table.String("daily_start", 5).Default("").Comment("HH:MM local time window start")
		table.String("daily_end", 5).Default("").Comment("HH:MM local time window end")
		table.String("weekdays", 32).Default("").Comment("weekly: 1=Mon..7=Sun csv")
		table.String("month_days", 64).Default("").Comment("monthly: day-of-month csv")
		table.UnsignedTinyInteger("month_day_start").Default(0).Comment("monthly range start 1-31")
		table.UnsignedTinyInteger("month_day_end").Default(0).Comment("monthly range end 1-31")
		table.String("year_start", 5).Default("").Comment("yearly MM-DD start")
		table.String("year_end", 5).Default("").Comment("yearly MM-DD end")
		table.String("timezone", 64).Default("Asia/Shanghai").Comment("IANA timezone")
		table.UnsignedTinyInteger("status").Default(0).Comment("0 pending 1 running 2 ended")
		table.Boolean("enabled").Default(true).Comment("manual disable")
		table.Timestamps()
		table.SoftDeletes()
		table.Index("enabled", "status")
		table.Index("schedule_type")
		table.Comment("Open-source demo: activity time windows")
	})
}

func (r *M20260920150000CreateDemoActivitiesTable) Down() error {
	return facades.Schema().DropIfExists("demo_activities")
}
