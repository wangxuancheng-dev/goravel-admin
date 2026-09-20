package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
)

// M20260920160000AddFlexibleScheduleToDemoActivities extends demo activity windows.
type M20260920160000AddFlexibleScheduleToDemoActivities struct{}

func (r *M20260920160000AddFlexibleScheduleToDemoActivities) Signature() string {
	return "20260920160000_add_flexible_schedule_to_demo_activities"
}

func (r *M20260920160000AddFlexibleScheduleToDemoActivities) Up() error {
	if !facades.Schema().HasTable("demo_activities") {
		return nil
	}
	type colSpec struct {
		name string
		add  func(schema.Blueprint)
	}
	cols := []colSpec{
		{"weekdays", func(table schema.Blueprint) {
			table.String("weekdays", 32).Default("").Comment("weekly: 1=Mon..7=Sun csv")
		}},
		{"month_days", func(table schema.Blueprint) {
			table.String("month_days", 64).Default("").Comment("monthly: day-of-month csv")
		}},
		{"month_day_start", func(table schema.Blueprint) {
			table.UnsignedTinyInteger("month_day_start").Default(0).Comment("monthly range start 1-31")
		}},
		{"month_day_end", func(table schema.Blueprint) {
			table.UnsignedTinyInteger("month_day_end").Default(0).Comment("monthly range end 1-31")
		}},
		{"year_start", func(table schema.Blueprint) {
			table.String("year_start", 5).Default("").Comment("yearly MM-DD start")
		}},
		{"year_end", func(table schema.Blueprint) {
			table.String("year_end", 5).Default("").Comment("yearly MM-DD end")
		}},
	}
	for _, col := range cols {
		if facades.Schema().HasColumn("demo_activities", col.name) {
			continue
		}
		adder := col.add
		if err := facades.Schema().Table("demo_activities", func(table schema.Blueprint) {
			adder(table)
		}); err != nil {
			return err
		}
	}
	return nil
}

func (r *M20260920160000AddFlexibleScheduleToDemoActivities) Down() error {
	if !facades.Schema().HasTable("demo_activities") {
		return nil
	}
	for _, col := range []string{"weekdays", "month_days", "month_day_start", "month_day_end", "year_start", "year_end"} {
		if !facades.Schema().HasColumn("demo_activities", col) {
			continue
		}
		c := col
		if err := facades.Schema().Table("demo_activities", func(table schema.Blueprint) {
			table.DropColumn(c)
		}); err != nil {
			return err
		}
	}
	return nil
}
