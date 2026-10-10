package models

import "github.com/goravel/framework/database/orm"

// PlatformPlan is a sellable / assignable entitlement bundle.
type PlatformPlan struct {
	orm.Model
	Code         string `gorm:"uniqueIndex;size:64;not null" json:"code"`
	Name         string `gorm:"size:100;not null" json:"name"`
	Description  string `gorm:"size:500" json:"description"`
	IsDefault    bool   `gorm:"default:false;index" json:"is_default"`
	IsPublic     bool   `gorm:"default:true" json:"is_public"`
	PriceMonthly int64  `gorm:"default:0" json:"price_monthly"` // minor units; 0 = free
	PriceYearly  int64  `gorm:"default:0" json:"price_yearly"`
	Currency     string `gorm:"size:8;default:CNY" json:"currency"`
	Status       uint8  `gorm:"default:1;index" json:"status"`
	Sort         int    `gorm:"default:0" json:"sort"`
}

func (PlatformPlan) TableName() string { return "platform_plans" }

// PlatformPlanEntitlement is one feature grant on a plan.
type PlatformPlanEntitlement struct {
	orm.Model
	PlanID     uint   `gorm:"index:idx_plan_feature,unique;not null" json:"plan_id"`
	FeatureKey string `gorm:"index:idx_plan_feature,unique;size:64;not null" json:"feature_key"`
	Value      string `gorm:"size:255;not null" json:"value"`
}

func (PlatformPlanEntitlement) TableName() string { return "platform_plan_entitlements" }
