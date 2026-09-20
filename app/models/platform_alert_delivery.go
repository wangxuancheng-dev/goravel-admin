package models

import (
	"time"

	"github.com/goravel/framework/database/orm"
)

const (
	PlatformAlertChannelWebhook = "webhook"
	PlatformAlertChannelMail    = "mail"

	PlatformAlertStatusPending = "pending"
	PlatformAlertStatusSuccess = "success"
	PlatformAlertStatusFailed  = "failed"
)

// PlatformAlertDelivery records landlord alert webhook/mail attempts.
type PlatformAlertDelivery struct {
	orm.Model
	Channel      string     `gorm:"size:32;not null;comment:webhook|mail" json:"channel"`
	Event        string     `gorm:"size:64;index;comment:event name" json:"event"`
	TenantID     *uint      `gorm:"index;comment:tenants.id" json:"tenant_id"`
	TenantCode   string     `gorm:"size:64;index;comment:tenant code" json:"tenant_code"`
	Op           string     `gorm:"size:64;comment:tenant op" json:"op"`
	Status       string     `gorm:"size:32;default:pending;index;comment:success|failed|pending" json:"status"`
	HTTPStatus   uint       `gorm:"default:0;comment:http status" json:"http_status"`
	TargetMasked string     `gorm:"size:255;comment:masked target" json:"target_masked"`
	Payload      string     `gorm:"type:text;comment:json payload" json:"payload"`
	ErrorMessage string     `gorm:"type:text;comment:error" json:"error_message"`
	Attempt      uint       `gorm:"default:1;comment:attempt count" json:"attempt"`
	DeliveredAt  *time.Time `gorm:"comment:delivered at" json:"delivered_at"`
}
