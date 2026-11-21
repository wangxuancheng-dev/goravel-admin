package models

import "github.com/goravel/framework/database/orm"

// AttachmentCategory 附件分类
type AttachmentCategory struct {
	orm.Model
	Name     string `gorm:"not null;size:50;comment:分类名称" json:"name"`
	Slug     string `gorm:"not null;size:50;uniqueIndex;comment:分类标识" json:"slug"`
	Status   uint8  `gorm:"comment:状态 1:启用 0:禁用" json:"status"`
	IsSystem uint8  `gorm:"default:0;comment:是否系统分类 1:是 0:否" json:"is_system"`
	Sort     int    `gorm:"default:0;comment:排序" json:"sort"`
	Remark   string `gorm:"size:500;comment:备注" json:"remark"`
	orm.SoftDeletes
}

const AttachmentCategorySlugUncategorized = "uncategorized"
