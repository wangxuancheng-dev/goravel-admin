package models

import (
	"github.com/goravel/framework/database/orm"
)

type Position struct {
	orm.Model
	Name   string  `gorm:"not null;size:50;comment:岗位名称" json:"name"`
	Code   string  `gorm:"size:50;comment:岗位编码" json:"code"`
	Status uint8   `gorm:"comment:状态 1:启用 0:禁用" json:"status"`
	Sort   int     `gorm:"default:0;comment:排序" json:"sort"`
	Remark string  `gorm:"size:500;comment:备注" json:"remark"`
	Admins []Admin `gorm:"foreignKey:PositionID" json:"-"`
}
