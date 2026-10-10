package models

import (
	"time"

	"github.com/goravel/framework/database/orm"
)

// TenantEntitlementSnapshot stores the materialized EffectiveSet JSON.
type TenantEntitlementSnapshot struct {
	orm.Model
	TenantID   uint      `gorm:"uniqueIndex;not null" json:"tenant_id"`
	Version    int64     `gorm:"default:1" json:"version"`
	PlanCode   string    `gorm:"size:64" json:"plan_code"`
	Payload    string    `gorm:"type:longtext" json:"payload"`
	ComputedAt time.Time `json:"computed_at"`
}

func (TenantEntitlementSnapshot) TableName() string { return "tenant_entitlement_snapshots" }

// TenantUsageCounter tracks limit consumption per period.
type TenantUsageCounter struct {
	orm.Model
	TenantID   uint   `gorm:"uniqueIndex:idx_usage_period;not null" json:"tenant_id"`
	FeatureKey string `gorm:"uniqueIndex:idx_usage_period;size:64;not null" json:"feature_key"`
	PeriodKey  string `gorm:"uniqueIndex:idx_usage_period;size:32;not null;default:lifetime" json:"period_key"`
	Used       int64  `gorm:"default:0" json:"used"`
}

func (TenantUsageCounter) TableName() string { return "tenant_usage_counters" }

// PlatformEntitlementAuditLog records platform entitlement mutations.
type PlatformEntitlementAuditLog struct {
	orm.Model
	AdminID  uint   `gorm:"index;default:0" json:"admin_id"`
	TenantID uint   `gorm:"index;default:0" json:"tenant_id"`
	Action   string `gorm:"size:64;not null;index" json:"action"`
	Detail   string `gorm:"type:text" json:"detail"`
}

func (PlatformEntitlementAuditLog) TableName() string { return "platform_entitlement_audit_logs" }
