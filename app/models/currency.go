package models

import (
	"github.com/goravel/framework/database/orm"
)

// Currency 货币表
type Currency struct {
	orm.Model
	Code          string  `gorm:"size:10;not null;uniqueIndex;comment:货币代码(如CNY,USD)" json:"code"`
	Name          string  `gorm:"size:50;not null;comment:货币名称(如人民币,美元)" json:"name"`
	Symbol        string  `gorm:"size:10;comment:货币符号(如¥,$)" json:"symbol"`
	Rate          float64 `gorm:"type:decimal(18,8);default:1;comment:汇率(相对于基准货币)" json:"rate"`
	DecimalPlaces int     `gorm:"comment:小数位数(如日元为0,人民币为2,虚拟货币为8)" json:"decimal_places"`
	IsDefault     bool    `gorm:"default:false;comment:是否默认货币" json:"is_default"`
	IsActive      bool    `gorm:"comment:是否启用" json:"is_active"`
	Sort          int     `gorm:"default:0;comment:排序" json:"sort"`
	Description   string  `gorm:"type:text;comment:描述" json:"description"`
	orm.SoftDeletes
}

// TableName 指定表名
func (Currency) TableName() string {
	return "currencies"
}
