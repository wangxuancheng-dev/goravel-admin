package models

import (
	"github.com/goravel/framework/database/orm"
)

// PlatformLoginLog records platform console login attempts (landlord DB).
type PlatformLoginLog struct {
	orm.Model
	AdminID   uint   `gorm:"index;comment:platform_admins.id" json:"admin_id"`
	Username  string `gorm:"size:50;index;comment:username" json:"username"`
	IP        string `gorm:"size:50;index;comment:ip" json:"ip"`
	UserAgent string `gorm:"size:500;comment:user agent" json:"user_agent"`
	Location  string `gorm:"size:100;comment:geo location" json:"location"`
	Status    uint8  `gorm:"index;comment:1 success 0 failed" json:"status"`
	Message   string `gorm:"size:255;comment:result message key" json:"message"`
	Request   string `gorm:"type:text;comment:sanitized request body" json:"request"`
}
