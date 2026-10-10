package models

import (
	"github.com/goravel/framework/database/orm"
)

type QuoteItem struct {
	orm.Model

	QuoteId uint `gorm:"column:quote_id" json:"quote_id" comment:"quotes.id"`

	Quote *Quote `gorm:"foreignKey:QuoteId" json:"quote"`

	ProductName string `gorm:"column:product_name" json:"product_name" comment:"product name"`

	Quantity int `gorm:"column:quantity" json:"quantity" comment:"qty"`

	UnitPrice float64 `gorm:"column:unit_price" json:"unit_price" comment:"unit price"`

	orm.SoftDeletes
}

func (QuoteItem) TableName() string {
	return "quote_items"
}
