package models

import (
	"github.com/goravel/framework/database/orm"
)

// PaymentMethod 支付方式表
type PaymentMethod struct {
	orm.Model
	orm.SoftDeletes
	Name        string `gorm:"size:50;not null;comment:支付方式名称(如微信支付,支付宝)" json:"name"`
	Code        string `gorm:"size:20;not null;uniqueIndex;comment:支付方式代码(如wechat,alipay)" json:"code"`
	Type        string `gorm:"size:20;not null;comment:支付类型(如wechat,alipay,qq,allinpay,lakala,paypal,apple,saobei)" json:"type"`
	Config      string `gorm:"type:text;comment:支付配置(JSON格式,存储密钥、证书等敏感信息)" json:"-"` // 不返回给前端
	IsActive    bool   `gorm:"comment:是否启用" json:"is_active"`
	Sort        int    `gorm:"default:0;comment:排序" json:"sort"`
	Description string `gorm:"type:text;comment:描述" json:"description"`
}

// TableName 指定表名
func (PaymentMethod) TableName() string {
	return "payment_methods"
}
