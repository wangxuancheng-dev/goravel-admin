package models

import (
	"encoding/json"
	"strings"

	"github.com/goravel/framework/database/orm"
)

// PlatformFeature is a landlord catalog entry for entitlement-managed capabilities.
type PlatformFeature struct {
	orm.Model
	Key             string `gorm:"uniqueIndex;size:64;not null" json:"key"`
	Name            string `gorm:"size:100;not null" json:"name"`
	Description     string `gorm:"size:500" json:"description"`
	Type            string `gorm:"size:16;not null;default:boolean" json:"type"` // boolean|limit|config
	DefaultValue    string `gorm:"size:255;default:false" json:"default_value"`
	MenuSlugsJSON   string `gorm:"column:menu_slugs;type:text" json:"-"`
	ChannelsJSON    string `gorm:"column:channels;type:text" json:"-"`
	IsTest          bool   `gorm:"default:false" json:"is_test"`
	RequiresInstall bool   `gorm:"default:false" json:"requires_install"`
	AlwaysOn        bool   `gorm:"default:false" json:"always_on"` // skip plan gating
	Status          uint8  `gorm:"default:1;index" json:"status"`  // 1 enabled in catalog
	Sort            int    `gorm:"default:0" json:"sort"`
}

func (PlatformFeature) TableName() string { return "platform_features" }

func (f *PlatformFeature) MenuSlugs() []string {
	return decodeStringSlice(f.MenuSlugsJSON)
}

func (f *PlatformFeature) SetMenuSlugs(slugs []string) {
	f.MenuSlugsJSON = encodeStringSlice(slugs)
}

func (f *PlatformFeature) Channels() []string {
	return decodeStringSlice(f.ChannelsJSON)
}

func (f *PlatformFeature) SetChannels(channels []string) {
	f.ChannelsJSON = encodeStringSlice(channels)
}

func decodeStringSlice(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	cleaned := make([]string, 0, len(out))
	for _, s := range out {
		s = strings.TrimSpace(s)
		if s != "" {
			cleaned = append(cleaned, s)
		}
	}
	return cleaned
}

func encodeStringSlice(items []string) string {
	cleaned := make([]string, 0, len(items))
	for _, s := range items {
		s = strings.TrimSpace(s)
		if s != "" {
			cleaned = append(cleaned, s)
		}
	}
	if len(cleaned) == 0 {
		return "[]"
	}
	b, err := json.Marshal(cleaned)
	if err != nil {
		return "[]"
	}
	return string(b)
}
