package models

import (
	"time"

	"github.com/goravel/framework/database/orm"
)

// Payment 支付记录表（按月分表）。跨分表时 id 仅表内唯一，业务主键请用 payment_no。
type Payment struct {
	orm.Model
	orm.SoftDeletes
	PaymentNo       string        `gorm:"index;not null;size:50;comment:支付单号" json:"payment_no"`
	OrderNo         string        `gorm:"index;not null;size:50;comment:订单号" json:"order_no"`
	PaymentMethodID uint          `gorm:"index;not null;comment:支付方式ID" json:"payment_method_id"`
	PaymentMethod   PaymentMethod `gorm:"foreignKey:PaymentMethodID" json:"payment_method,omitempty"`
	UserID          uint          `gorm:"index;not null;comment:用户ID" json:"user_id"`
	Amount          float64       `gorm:"type:decimal(10,2);not null;comment:支付金额" json:"amount"`
	Status          string        `gorm:"size:20;default:'pending';comment:支付状态 pending:待支付 paid:已支付 failed:支付失败 cancelled:已取消" json:"status"`
	ThirdPartyNo    string        `gorm:"size:100;comment:第三方支付单号" json:"third_party_no"`
	PayTime         *time.Time    `gorm:"comment:支付时间" json:"pay_time"`
	FailReason      string        `gorm:"type:text;comment:失败原因" json:"fail_reason"`
	NotifyData      string        `gorm:"type:text;comment:回调通知数据(JSON格式)" json:"-"` // 不返回给前端
	Remark          string        `gorm:"type:text;comment:备注" json:"remark"`
}

// Payment status values (stored as string).
const (
	PaymentStatusPending   = "pending"
	PaymentStatusPaid      = "paid"
	PaymentStatusFailed    = "failed"
	PaymentStatusCancelled = "cancelled"
)

// TableName 指定表名
func (Payment) TableName() string {
	return "payments"
}
