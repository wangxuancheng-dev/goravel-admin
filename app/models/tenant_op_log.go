package models

import (
	"time"

	"github.com/goravel/framework/database/orm"
)

// TenantOpLog records platform tenant maintenance history (landlord DB).
type TenantOpLog struct {
	orm.Model
	TenantID  uint       `gorm:"index;not null;comment:tenants.id" json:"tenant_id"`
	Code      string     `gorm:"size:64;index;comment:tenant code snapshot" json:"code"`
	Op        string     `gorm:"size:32;index;not null;comment:migrate|seed|backup|restore" json:"op"`
	Status    string     `gorm:"size:32;index;not null;comment:queued|running|success|failed" json:"status"`
	Message   string     `gorm:"type:text;comment:result or error" json:"message"`
	StartedAt *time.Time `gorm:"comment:op start" json:"started_at"`
	FinishedAt *time.Time `gorm:"comment:op end" json:"finished_at"`
}
