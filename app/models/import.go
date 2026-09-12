package models

import "github.com/goravel/framework/database/orm"

// Import 异步导入记录
type Import struct {
	orm.Model
	AdminID       uint   `gorm:"index;comment:管理员ID"`
	Admin         Admin  `gorm:"foreignKey:AdminID"`
	Type          string `gorm:"size:50;index;comment:导入类型 orders:订单导入"`
	Status        uint8  `gorm:"default:0;comment:状态 0:处理中 1:成功 2:失败"`
	Disk          string `gorm:"size:50;comment:存储驱动"`
	Path          string `gorm:"size:255;comment:源文件路径"`
	TotalRows     int    `gorm:"default:0;comment:总行数"`
	SuccessRows   int    `gorm:"default:0;comment:成功行数"`
	FailedRows    int    `gorm:"default:0;comment:失败行数"`
	ErrorFilePath string `gorm:"size:255;comment:失败行 CSV 路径"`
	ErrorMsg      string `gorm:"type:text;comment:错误信息"`
}

const (
	ImportStatusProcessing uint8 = 0
	ImportStatusSuccess    uint8 = 1
	ImportStatusFailed     uint8 = 2
)

const (
	ImportTypeOrders string = "orders"
)
