package models

import (
	"time"

	"github.com/goravel/framework/database/orm"
)

// TenantEntitlementOverride is a manual grant that wins over plan/addon.
type TenantEntitlementOverride struct {
	orm.Model
	TenantID    uint       `gorm:"uniqueIndex:idx_tenant_feature_override;not null" json:"tenant_id"`
	FeatureKey  string     `gorm:"uniqueIndex:idx_tenant_feature_override;size:64;not null" json:"feature_key"`
	Value       string     `gorm:"size:255;not null" json:"value"`
	Pinned      bool       `gorm:"default:true" json:"pinned"` // survive plan reconcile
	ExpiresAt   *time.Time `gorm:"index" json:"expires_at"`
	Reason      string     `gorm:"size:500" json:"reason"`
	CreatedBy   uint       `gorm:"default:0" json:"created_by"`
	InstalledAt *time.Time `json:"installed_at"`
}

func (TenantEntitlementOverride) TableName() string { return "tenant_entitlement_overrides" }
