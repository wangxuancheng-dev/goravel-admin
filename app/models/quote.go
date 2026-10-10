package models

import (
	"github.com/goravel/framework/database/orm"
)

type Quote struct {
	orm.Model

	QuoteNo string `gorm:"column:quote_no" json:"quote_no" comment:"quote number"`

	CustomerName string `gorm:"column:customer_name" json:"customer_name" comment:"customer name"`

	Status uint8 `gorm:"column:status" json:"status" comment:"0 draft 1 confirmed"`

	Remark string `gorm:"column:remark" json:"remark" comment:"remark"`

	Details []QuoteItem `gorm:"foreignKey:QuoteId" json:"details"`

	orm.SoftDeletes
}

func (Quote) TableName() string {
	return "quotes"
}
