package models

import (
	"github.com/goravel/framework/database/orm"
)

// PlatformOperationLog records platform console write operations (landlord DB).
type PlatformOperationLog struct {
	orm.Model
	AdminID   uint   `gorm:"index;comment:platform_admins.id" json:"admin_id"`
	Username  string `gorm:"size:50;index;comment:username snapshot" json:"username"`
	Method    string `gorm:"size:10;index;comment:http method" json:"method"`
	Path      string `gorm:"size:255;index;comment:request path" json:"path"`
	Title     string `gorm:"size:255;comment:operation title" json:"title"`
	IP        string `gorm:"size:50;comment:ip" json:"ip"`
	UserAgent string `gorm:"size:500;comment:user agent" json:"user_agent"`
	Request   string `gorm:"type:text;comment:sanitized request body" json:"request"`
	Status    uint8  `gorm:"default:1;index;comment:1 success 0 failed" json:"status"`
	ErrorMsg  string `gorm:"type:text;comment:error message" json:"error_msg"`
	Duration  int    `gorm:"comment:duration ms" json:"duration"`
}
