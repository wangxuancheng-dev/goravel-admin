package models

import (
	"time"

	"github.com/goravel/framework/database/orm"
)

// TenantSubscription links a tenant to a plan (landlord DB).
type TenantSubscription struct {
	orm.Model
	TenantID    uint       `gorm:"index;not null" json:"tenant_id"`
	PlanID      uint       `gorm:"index;not null" json:"plan_id"`
	Status      string     `gorm:"size:32;not null;index;default:active" json:"status"`
	BillingCycle string    `gorm:"size:16;default:manual" json:"billing_cycle"` // monthly|yearly|manual
	StartsAt    *time.Time `json:"starts_at"`
	EndsAt      *time.Time `json:"ends_at"`
	TrialEndsAt *time.Time `json:"trial_ends_at"`
	ExternalRef string     `gorm:"size:128" json:"external_ref"`
	Note        string     `gorm:"size:500" json:"note"`
}

func (TenantSubscription) TableName() string { return "tenant_subscriptions" }
