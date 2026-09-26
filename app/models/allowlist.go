package models

import (
	"github.com/goravel/framework/database/orm"
)

// Allowlist is an IP allowlist entry. When any row has status=1, only matching
// client IPs may access the admin API (empty enabled set = allow all).
type Allowlist struct {
	orm.Model
	IP     string `gorm:"index;not null;size:500;comment:IP or CIDR / range, comma-separated" json:"ip"`
	Remark string `gorm:"size:500;comment:remark" json:"remark"`
	Status uint8  `gorm:"index;default:1;comment:1 enabled 0 disabled" json:"status"`
}
