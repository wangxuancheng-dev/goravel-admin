package models

import (
	"time"

	"github.com/goravel/framework/database/orm"
)

// FlexibleSchedule is a landlord-row cron job with a code whitelist handler.
// tenant_id=0 means global (handler decides fan-out); >0 scopes to one tenant.
type FlexibleSchedule struct {
	orm.Model
	Name           string     `gorm:"column:name;size:100;not null;comment:display name" json:"name"`
	Handler        string     `gorm:"column:handler;size:64;not null;index;comment:whitelist handler key" json:"handler"`
	CronExpr       string     `gorm:"column:cron_expr;size:64;not null;comment:5-field cron min hour dom month dow" json:"cron_expr"`
	Timezone       string     `gorm:"column:timezone;size:64;default:UTC;comment:IANA timezone for cron eval" json:"timezone"`
	TenantID       uint       `gorm:"column:tenant_id;default:0;index;comment:0=global else tenants.id" json:"tenant_id"`
	Enabled        bool       `gorm:"column:enabled;default:true;index;comment:enabled" json:"enabled"`
	LastRunAt      *time.Time `gorm:"column:last_run_at;comment:last execution time UTC" json:"last_run_at"`
	LastStatus     string     `gorm:"column:last_status;size:16;default:never;comment:never|success|failed|skipped" json:"last_status"`
	LastError      string     `gorm:"column:last_error;type:text;comment:last error" json:"last_error"`
	LastOutput     string     `gorm:"column:last_output;type:text;comment:last output truncated" json:"last_output"`
	LastDurationMs int64      `gorm:"column:last_duration_ms;default:0;comment:last duration ms" json:"last_duration_ms"`
	LastSlot       string     `gorm:"column:last_slot;size:32;comment:dedupe slot YYYY-MM-DDTHH:MM in timezone" json:"last_slot"`
}

func (FlexibleSchedule) TableName() string {
	return "flexible_schedules"
}
