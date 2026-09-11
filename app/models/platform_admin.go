package models

import (
	"github.com/goravel/framework/database/orm"
)

const (
	PlatformAdminStatusActive   uint8 = 1
	PlatformAdminStatusDisabled uint8 = 0

	TokenableTypePlatformAdmin = "platform_admin"
)

// PlatformAdmin 平台控制台账号（仅存在于平台库 DB_*）。
type PlatformAdmin struct {
	orm.Model
	Username string `gorm:"uniqueIndex;size:50;not null;comment:用户名" json:"username"`
	Password string `gorm:"size:255;not null;comment:密码" json:"-"`
	Name     string `gorm:"size:100;comment:显示名" json:"name"`
	Status   uint8  `gorm:"default:1;index;comment:1启用 0禁用" json:"status"`
	orm.SoftDeletes
}
