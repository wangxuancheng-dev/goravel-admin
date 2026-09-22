package models

import (
	"time"

	"github.com/goravel/framework/database/orm"
)

// FlexibleSchedule is a landlord-row cron job with a code whitelist handler.
// tenant_id scopes the row to one tenant admin (0 when tenancy is off).
type FlexibleSchedule struct {
	orm.Model
	Name           string     `gorm:"column:name;size:100;not null;comment:display name" json:"name"`
	Handler        string     `gorm:"column:handler;size:64;not null;uniqueIndex:uk_flexible_handler_tenant;comment:whitelist handler key" json:"handler"`
	CronExpr       string     `gorm:"column:cron_expr;size:64;not null;comment:5-field cron min hour dom month dow" json:"cron_expr"`
	Timezone       string     `gorm:"column:timezone;size:64;default:UTC;comment:IANA timezone for cron eval" json:"timezone"`
	TenantID       uint       `gorm:"column:tenant_id;default:0;uniqueIndex:uk_flexible_handler_tenant;comment:0=no tenancy else tenants.id" json:"tenant_id"`
	Payload        string     `gorm:"column:payload;type:text;comment:handler options JSON object" json:"payload"`
	Enabled        bool       `gorm:"column:enabled;default:true;index;comment:enabled" json:"enabled"`
	LastRunAt      *time.Time `gorm:"column:last_run_at;comment:last execution time UTC" json:"last_run_at"`
	LastStatus     string     `gorm:"column:last_status;size:16;default:never;comment:never|success|failed|skipped" json:"last_status"`
	LastError      string     `gorm:"column:last_error;type:text;comment:last error" json:"last_error"`
	LastOutput     string     `gorm:"column:last_output;type:text;comment:last output truncated" json:"last_output"`
	LastDurationMs int64      `gorm:"column:last_duration_ms;default:0;comment:last duration ms" json:"last_duration_ms"`
	LastSlot       string     `gorm:"column:last_slot;size:32;comment:dedupe slot YYYY-MM-DDTHH:MM in timezone" json:"last_slot"`
	NextRunAt      *time.Time `gorm:"column:next_run_at;index;comment:next due UTC for tick query" json:"next_run_at"`
}

func (FlexibleSchedule) TableName() string {
	return "flexible_schedules"
}
