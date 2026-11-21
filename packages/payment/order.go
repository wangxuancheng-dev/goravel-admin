package payment

import "time"

// Status values for Order.Status (aligned with common payment records).
const (
	StatusPending   = "pending"
	StatusPaid      = "paid"
	StatusFailed    = "failed"
	StatusCancelled = "cancelled"
)

// Order is a host-agnostic payment order snapshot passed into gateway drivers.
// Host apps map their own payment model into this DTO before Create/Query.
type Order struct {
	PaymentNo    string
	Amount       float64
	Status       string
	Remark       string
	ThirdPartyNo string
	PayTime      *time.Time
	// PayerID identifies the payer at the provider when a trade type needs it
	// (e.g. the WeChat openid for jsapi). It belongs to the order, not the merchant config.
	PayerID string
}
