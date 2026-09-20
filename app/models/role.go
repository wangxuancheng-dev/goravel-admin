package models

import (
	"github.com/goravel/framework/database/orm"
)

// Data scope values on Role (RuoYi-compatible).
const (
	DataScopeAll          uint8 = 1 // 全部数据
	DataScopeCustom       uint8 = 2 // 自定义部门
	DataScopeDept         uint8 = 3 // 本部门
	DataScopeDeptAndChild uint8 = 4 // 本部门及以下
	DataScopeSelf         uint8 = 5 // 仅本人
)

type Role struct {
	orm.Model
	Name        string       `gorm:"uniqueIndex;not null;size:50;comment:角色名称" json:"name"`
	Slug        string       `gorm:"uniqueIndex;not null;size:50;comment:角色标识" json:"slug"`
	Description string       `gorm:"size:255;comment:角色描述" json:"description"`
	Status      uint8        `gorm:"default:1;comment:状态 1:启用 0:禁用" json:"status"`
	Sort        int          `gorm:"default:0;comment:排序" json:"sort"`
	DataScope   uint8        `gorm:"default:1;comment:数据范围 1全部 2自定义 3本部门 4本部门及以下 5仅本人" json:"data_scope"`
	Admins      []Admin      `gorm:"many2many:admin_role;comment:管理员" json:"-"`
	Permissions []Permission `gorm:"many2many:role_permission;comment:权限" json:"permissions,omitempty"`
	Menus       []Menu       `gorm:"many2many:role_menu;comment:菜单" json:"menus,omitempty"`
	Departments []Department `gorm:"many2many:role_department;comment:自定义数据权限部门" json:"departments,omitempty"`
}
