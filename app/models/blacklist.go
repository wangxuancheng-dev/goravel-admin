package models

import (
	"github.com/goravel/framework/database/orm"
)

type Blacklist struct {
	orm.Model
	IP     string `gorm:"index;not null;size:500;comment:IP地址或IP段，支持单个IP、CIDR格式、IP范围，多个用逗号分隔" json:"ip"`
	Remark string `gorm:"size:500;comment:备注" json:"remark"`
	Status uint8  `gorm:"index;comment:状态 1:启用 0:禁用" json:"status"`
}
