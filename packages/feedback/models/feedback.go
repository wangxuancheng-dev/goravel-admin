package models

import (
	"github.com/goravel/framework/database/orm"
)

// Feedback is a tenant-scoped feedback row (one table per tenant DB when tenancy is on).
type Feedback struct {
	orm.Model
	Contact   string `gorm:"column:contact;size:128" json:"contact"`
	Content   string `gorm:"column:content;type:text" json:"content"`
	Status    uint8  `gorm:"column:status;default:0" json:"status"` // 0=open 1=done
	CreatedBy uint   `gorm:"column:created_by;default:0" json:"created_by"`
	orm.SoftDeletes
}

func (Feedback) TableName() string {
	return "feedbacks"
}
