package models

import (
	"time"

	"github.com/goravel/framework/database/orm"
)

const (
	TenantDomainStatusPending  = "pending"
	TenantDomainStatusVerified = "verified"
	TenantDomainStatusActive   = "active"
	TenantDomainStatusDisabled = "disabled"

	TenantDomainSSLEdge        = "edge"
	TenantDomainSSLCustomerCDN = "customer_cdn"

	TenantDomainVerifyDNSTXT = "dns_txt"
)

// TenantDomain is a vanity host mapped to a landlord tenant (not row-level tenancy).
type TenantDomain struct {
	orm.Model
	TenantID       uint       `gorm:"index;not null;comment:tenants.id" json:"tenant_id"`
	Host           string     `gorm:"size:255;uniqueIndex;not null;comment:normalized custom host" json:"host"`
	IsPrimary      bool       `gorm:"default:false;comment:primary vanity host" json:"is_primary"`
	Status         string     `gorm:"size:32;index;default:pending;comment:pending|verified|active|disabled" json:"status"`
	SSLMode        string     `gorm:"size:32;index;default:edge;comment:edge|customer_cdn" json:"ssl_mode"`
	VerifyType     string     `gorm:"size:32;default:dns_txt;comment:dns_txt" json:"verify_type"`
	VerifyToken    string     `gorm:"size:64;comment:dns txt token" json:"verify_token"`
	VerifiedAt     *time.Time `gorm:"comment:verified at" json:"verified_at"`
	LastCheckAt    *time.Time `gorm:"comment:last verify attempt" json:"last_check_at"`
	LastCheckError string     `gorm:"type:text;comment:last verify error" json:"last_check_error"`
	orm.SoftDeletes
}
