package models

import "github.com/goravel/framework/database/orm"

// Attachment 附件表
// 用于管理所有上传的文件，支持分片上传和断点续传
type Attachment struct {
	orm.Model
	AdminID     uint                `gorm:"index;comment:管理员ID"`
	Admin       *Admin              `gorm:"foreignKey:AdminID"`
	CategoryID  uint                `gorm:"index;comment:分类ID"`
	Category    *AttachmentCategory `gorm:"foreignKey:CategoryID"`
	Disk        string              `gorm:"size:50;comment:存储驱动"`
	Path        string              `gorm:"size:500;comment:文件路径"`
	Filename    string              `gorm:"size:255;comment:原始文件名"`
	DisplayName string              `gorm:"size:255;index;comment:显示名称"`
	Extension   string              `gorm:"size:20;comment:文件后缀"`
	MimeType    string              `gorm:"size:100;comment:MIME类型"`
	Size        int64               `gorm:"comment:文件大小(字节)"`
	Status      uint8               `gorm:"comment:状态 1:成功 0:失败 2:上传中"`
	FileType    string              `gorm:"size:20;comment:文件类型 image/video/document/other"`
	IsPublic    uint8               `gorm:"index;comment:是否公开 1:公开 0:私有"`
	ChunkID     string              `gorm:"size:100;index;comment:分片上传ID（用于断点续传）"`
}
