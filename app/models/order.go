package models

import (
	"time"

	"github.com/goravel/framework/database/orm"
)

// Order 订单主表（按月分表）。跨分表时 id 仅表内唯一，业务主键请用 order_no。
type Order struct {
	orm.Model
	orm.SoftDeletes         // 软删除支持
	OrderNo         string  `gorm:"index;not null;size:50;comment:订单号"`
	UserID          uint    `gorm:"index;not null;comment:用户ID"`
	Amount          float64 `gorm:"type:decimal(10,2);not null;comment:订单金额"`
	Status          string  `gorm:"size:20;default:'pending';comment:订单状态 pending:待支付 paid:已支付 cancelled:已取消"`
	Remark          string  `gorm:"type:text;comment:备注"`
	ExpireAt        *time.Time `gorm:"column:expire_at;comment:pending auto-cancel deadline UTC" json:"expire_at"`
	// CreatedAt 字段由 orm.Model 提供，用于分表
}

// Order status values (stored as string).
const (
	OrderStatusPending   = "pending"
	OrderStatusPaid      = "paid"
	OrderStatusCancelled = "cancelled"
)

// TableName 指定表名（分表时会自动添加后缀）
func (Order) TableName() string {
	return "orders"
}

// OrderDetail 订单详情表
type OrderDetail struct {
	orm.Model
	orm.SoftDeletes         // 软删除支持
	OrderID         uint    `gorm:"index;not null;comment:订单ID"`
	ProductID       uint    `gorm:"index;not null;comment:商品ID"`
	ProductName     string  `gorm:"size:200;not null;comment:商品名称"`
	Price           float64 `gorm:"type:decimal(10,2);not null;comment:单价"`
	Quantity        int     `gorm:"not null;comment:数量"`
	Subtotal        float64 `gorm:"type:decimal(10,2);not null;comment:小计"`
	// CreatedAt 字段由 orm.Model 提供，用于分表
}

// TableName 指定表名（分表时会自动添加后缀）
func (OrderDetail) TableName() string {
	return "order_details"
}
