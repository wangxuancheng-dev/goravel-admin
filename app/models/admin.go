package models

import (
	"github.com/goravel/framework/database/orm"
)

type Admin struct {
	orm.Model
	Username           string     `gorm:"not null;size:50;comment:用户名" json:"username"`
	Password           string     `gorm:"not null;size:255;comment:密码" json:"-"` // 使用 json:"-" 隐藏密码字段
	Nickname           string     `gorm:"size:50;comment:昵称" json:"nickname"`
	Avatar             string     `gorm:"size:255;comment:头像" json:"avatar"`
	Email              string     `gorm:"size:100;comment:邮箱" json:"email"`
	Phone              string     `gorm:"size:20;comment:手机号" json:"phone"`
	GoogleSecret       string     `gorm:"size:255;comment:谷歌验证码密钥" json:"-"` // 使用 json:"-" 隐藏密钥字段
	MustChangePassword uint8      `gorm:"default:0;comment:是否必须修改密码 1:是 0:否" json:"must_change_password"`
	Status             uint8      `gorm:"default:1;comment:状态 1:启用 0:禁用" json:"status"`
	DepartmentID       uint       `gorm:"index;comment:部门ID" json:"department_id"`
	Department         Department `gorm:"foreignKey:DepartmentID" json:"department"`
	PositionID         uint       `gorm:"index;comment:岗位ID" json:"position_id"`
	Position           Position   `gorm:"foreignKey:PositionID" json:"position"`
	Roles              []Role     `gorm:"many2many:admin_role;comment:角色" json:"roles"`
	orm.SoftDeletes
}
