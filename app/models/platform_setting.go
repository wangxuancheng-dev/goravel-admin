package models

import "github.com/goravel/framework/database/orm"

// PlatformSetting is a landlord key/value setting for the platform console (e.g. captcha type).
type PlatformSetting struct {
	orm.Model
	Key   string `gorm:"column:key;size:64;uniqueIndex;not null" json:"key"`
	Value string `gorm:"column:value;type:text" json:"value"`
}

func (PlatformSetting) TableName() string { return "platform_settings" }
