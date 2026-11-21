package models

import (
	"github.com/goravel/framework/database/orm"
)

type Permission struct {
	orm.Model
	Name        string `gorm:"uniqueIndex;not null;size:50;comment:权限名称"`
	Slug        string `gorm:"uniqueIndex;not null;size:100;comment:权限标识"`
	Method      string `gorm:"size:10;comment:请求方法 GET,POST,PUT,DELETE等"`
	Path        string `gorm:"size:255;comment:路由路径"`
	Description string `gorm:"size:255;comment:权限描述"`
	Status      uint8  `gorm:"comment:状态 1:启用 0:禁用"`
	Sort        int    `gorm:"default:0;comment:排序"`
	MenuID      uint   `gorm:"index;default:0;comment:关联菜单ID"`
	Menu        Menu   `gorm:"foreignKey:MenuID"`
	Roles       []Role `gorm:"many2many:role_permission;comment:角色"`
}
