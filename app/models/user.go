package models

import (
	"time"

	"github.com/goravel/framework/database/orm"
)

// User 用户表（普通用户，区别于管理员）
type User struct {
	orm.Model
	Username    string     `gorm:"not null;size:50;uniqueIndex;comment:用户名" json:"username"`
	Password    string     `gorm:"not null;size:255;comment:密码" json:"-"` // 使用 json:"-" 隐藏密码字段
	Nickname    string     `gorm:"size:50;comment:昵称" json:"nickname"`
	Avatar      string     `gorm:"size:255;comment:头像" json:"avatar"`
	Email       string     `gorm:"size:100;index;comment:邮箱" json:"email"`
	Phone       string     `gorm:"size:20;index;comment:手机号" json:"phone"`
	Balance     float64    `gorm:"type:decimal(18,8);default:0;comment:当前余额" json:"balance"`
	CurrencyID  uint       `gorm:"index;comment:货币ID" json:"currency_id"`
	Currency    *Currency  `gorm:"foreignKey:CurrencyID" json:"currency,omitempty"`
	Status      uint8      `gorm:"comment:状态 1:启用 0:禁用" json:"status"`
	LastLoginAt *time.Time `gorm:"comment:最后登录时间" json:"last_login_at"`
	orm.SoftDeletes
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}
