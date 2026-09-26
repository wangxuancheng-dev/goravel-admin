package models

import "github.com/goravel/framework/database/orm"

// Export 导出记录表
// 用于记录所有导出文件的信息，便于在后台进行统一管理
type Export struct {
	orm.Model
	AdminID   uint   `gorm:"index;comment:管理员ID"`
	Admin     Admin  `gorm:"foreignKey:AdminID"`
	Type      string `gorm:"size:50;index;comment:导出类型 orders:订单导出 admins:管理员导出"`
	Disk      string `gorm:"size:50;comment:存储驱动"`
	Path      string `gorm:"size:255;comment:文件路径"`
	Filename  string `gorm:"size:255;comment:文件名"`
	Extension string `gorm:"size:20;comment:文件后缀"`
	Size      int64  `gorm:"comment:文件大小(字节)"`
	Status    uint8  `gorm:"default:0;comment:状态 0:处理中 1:成功 2:失败"`
	ErrorMsg  string `gorm:"type:text;comment:错误信息"`
}

// 导出状态常量
const (
	ExportStatusProcessing uint8 = 0 // 处理中
	ExportStatusSuccess    uint8 = 1 // 成功
	ExportStatusFailed     uint8 = 2 // 失败
)

// 导出类型常量
const (
	ExportTypeOrders                string = "orders"                  // 订单导出
	ExportTypeAdmins                string = "admins"                  // 管理员导出
	ExportTypePayments              string = "payments"                // 支付记录导出
	ExportTypeUsers                 string = "users"                   // 用户导出
	ExportTypeOperationLogsArchive  string = "operation_logs_archive"  // 操作日志归档
	ExportTypeLoginLogsArchive      string = "login_logs_archive"      // 登录日志归档
	ExportTypeOperationLogsExport   string = "operation_logs_export"   // 操作日志导出（不删除）
)
