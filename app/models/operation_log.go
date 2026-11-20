package models

import (
	"github.com/goravel/framework/database/orm"
)

type OperationLog struct {
	orm.Model
	AdminID   uint   `gorm:"index;comment:管理员ID" json:"admin_id"`
	Admin     Admin  `gorm:"foreignKey:AdminID" json:"admin"`
	TraceID   string `gorm:"size:120;index;comment:链路ID" json:"trace_id"`
	Method    string `gorm:"size:10;comment:请求方法" json:"method"`
	Path      string `gorm:"size:255;comment:请求路径" json:"path"`
	Title     string `gorm:"size:255;comment:操作标题" json:"title"`
	IP        string `gorm:"size:50;comment:IP地址" json:"ip"`
	UserAgent string `gorm:"size:500;comment:用户代理" json:"user_agent"`
	Request   string `gorm:"type:text;comment:请求参数" json:"request"`
	Response  string `gorm:"type:text;comment:响应数据" json:"response"`
	Changes   string `gorm:"type:text;comment:变更详情(JSON diff)" json:"changes"`
	Status    uint8  `gorm:"default:1;comment:状态 1:成功 0:失败" json:"status"`
	ErrorMsg  string `gorm:"type:text;comment:错误信息" json:"error_msg"`
	Duration  int    `gorm:"comment:耗时(毫秒)" json:"duration"`
	// Impersonator is "<platform_username>#<platform_admin_id>" when a platform account
	// acted through "sign in as tenant admin"; empty for the admin's own actions.
	Impersonator string `gorm:"size:120;default:'';comment:平台代管账号" json:"impersonator"`
}
