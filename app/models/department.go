package models

import (
	"github.com/goravel/framework/database/orm"
)

type Department struct {
	orm.Model
	ParentID uint         `gorm:"index;default:0;comment:父级ID" json:"parent_id"`
	Name     string       `gorm:"not null;size:50;comment:部门名称" json:"name"`
	Code     string       `gorm:"size:50;comment:部门编码" json:"code"`
	Leader   string       `gorm:"size:50;comment:负责人" json:"leader"`
	Phone    string       `gorm:"size:20;comment:联系电话" json:"phone"`
	Email    string       `gorm:"size:100;comment:邮箱" json:"email"`
	Status   uint8        `gorm:"default:1;comment:状态 1:启用 0:禁用" json:"status"`
	Sort     int          `gorm:"default:0;comment:排序" json:"sort"`
	Remark   string       `gorm:"size:500;comment:备注" json:"remark"`
	Children []Department `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	Admins   []Admin      `gorm:"foreignKey:DepartmentID" json:"-"`
}