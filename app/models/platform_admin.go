package models

import (
	"strings"

	"github.com/goravel/framework/database/orm"
)

const (
	PlatformAdminStatusActive   uint8 = 1
	PlatformAdminStatusDisabled uint8 = 0

	PlatformAdminRoleOwner  = "owner"
	PlatformAdminRoleViewer = "viewer"

	TokenableTypePlatformAdmin = "platform_admin"
)

// PlatformAdmin is a landlord console account (platform DB only).
type PlatformAdmin struct {
	orm.Model
	Username string `gorm:"uniqueIndex;size:50;not null;comment:username" json:"username"`
	Password string `gorm:"size:255;not null;comment:password hash" json:"-"`
	Name     string `gorm:"size:100;comment:display name" json:"name"`
	Role     string `gorm:"size:32;default:owner;index;comment:owner|viewer" json:"role"`
	Status   uint8  `gorm:"default:1;index;comment:1 active 0 disabled" json:"status"`
	orm.SoftDeletes
}

// NormalizePlatformAdminRole returns owner|viewer (empty => owner).
func NormalizePlatformAdminRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case PlatformAdminRoleViewer:
		return PlatformAdminRoleViewer
	default:
		return PlatformAdminRoleOwner
	}
}

// IsOwner reports full platform ops permission (empty role treated as owner).
func (a *PlatformAdmin) IsOwner() bool {
	if a == nil {
		return false
	}
	return NormalizePlatformAdminRole(a.Role) == PlatformAdminRoleOwner
}

// IsViewer reports read-only platform console access.
func (a *PlatformAdmin) IsViewer() bool {
	if a == nil {
		return false
	}
	return NormalizePlatformAdminRole(a.Role) == PlatformAdminRoleViewer
}
