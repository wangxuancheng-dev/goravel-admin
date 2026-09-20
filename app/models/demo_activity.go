package models

import (
	"time"

	"github.com/goravel/framework/database/orm"
)

// DemoActivity is an open-source sample for flexible time-window scheduling.
// Not a production marketing module.
type DemoActivity struct {
	orm.Model
	Title         string     `gorm:"column:title;size:120;not null" json:"title"`
	ScheduleType  string     `gorm:"column:schedule_type;size:16;not null;default:once" json:"schedule_type"`
	StartAt       *time.Time `gorm:"column:start_at" json:"start_at"`
	EndAt         *time.Time `gorm:"column:end_at" json:"end_at"`
	DailyStart    string     `gorm:"column:daily_start;size:5;default:''" json:"daily_start"`
	DailyEnd      string     `gorm:"column:daily_end;size:5;default:''" json:"daily_end"`
	Weekdays      string     `gorm:"column:weekdays;size:32;default:''" json:"weekdays"`             // weekly: "1,2,3,4,5" (1=Mon..7=Sun)
	MonthDays     string     `gorm:"column:month_days;size:64;default:''" json:"month_days"`         // monthly: "1,15,28"
	MonthDayStart uint8      `gorm:"column:month_day_start;default:0" json:"month_day_start"`         // monthly range start (1-31)
	MonthDayEnd   uint8      `gorm:"column:month_day_end;default:0" json:"month_day_end"`             // monthly range end (1-31)
	YearStart     string     `gorm:"column:year_start;size:5;default:''" json:"year_start"`           // yearly: MM-DD
	YearEnd       string     `gorm:"column:year_end;size:5;default:''" json:"year_end"`               // yearly: MM-DD
	Timezone      string     `gorm:"column:timezone;size:64;not null;default:Asia/Shanghai" json:"timezone"`
	Status        uint8      `gorm:"column:status;not null;default:0" json:"status"`
	Enabled       bool       `gorm:"column:enabled;not null;default:1" json:"enabled"`
	orm.SoftDeletes
}

const (
	DemoActivityScheduleOnce    = "once"
	DemoActivityScheduleDaily   = "daily"
	DemoActivityScheduleWeekly  = "weekly"
	DemoActivityScheduleMonthly = "monthly"
	DemoActivityScheduleYearly  = "yearly"

	DemoActivityStatusPending uint8 = 0
	DemoActivityStatusRunning uint8 = 1
	DemoActivityStatusEnded   uint8 = 2
)

func (DemoActivity) TableName() string {
	return "demo_activities"
}
