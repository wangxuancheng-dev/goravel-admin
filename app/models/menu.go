package models

import (
	"github.com/goravel/framework/database/orm"
)

type Menu struct {
	orm.Model
	ParentID   uint   `gorm:"index;default:0;comment:父级ID"`
	Title      string `gorm:"not null;size:50;comment:菜单标题"`
	Slug       string `gorm:"uniqueIndex;size:50;comment:菜单标识"`
	Icon       string `gorm:"size:50;comment:图标"`
	Path       string `gorm:"size:1000;comment:路由路径"`
	Component  string `gorm:"size:255;comment:组件路径"`
	Permission string `gorm:"size:100;comment:权限标识"`
	Type       uint8  `gorm:"default:1;comment:类型 1:目录 2:菜单 3:按钮"`
	Status     uint8  `gorm:"comment:状态 1:启用 0:禁用"`
	Sort       int    `gorm:"default:0;comment:排序"`
	IsHidden   uint8  `gorm:"default:0;comment:是否隐藏 1:是 0:否"`
	LinkType   uint8  `gorm:"default:1;comment:链接类型 1:内部页面 2:外部链接"`
	OpenType   uint8  `gorm:"default:1;comment:打开方式 1:iframe嵌套 2:新窗口打开"`
	NoCache    uint8  `gorm:"default:0;comment:是否缓存 1:否-每次进页面刷新 0:是"`
	Children   []Menu `gorm:"foreignKey:ParentID"`
	Roles      []Role `gorm:"many2many:role_menu;comment:角色"`
}
