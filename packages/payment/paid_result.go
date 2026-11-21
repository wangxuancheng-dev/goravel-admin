package payment

import (
	"fmt"
	"time"
)

// PaidResult is the normalized outcome after a gateway verifies a successful payment.
// Drivers only verify + map fields; the host applies DB writes (orders/payments).
type PaidResult struct {
	PaymentNo    string
	ThirdPartyNo string
	PayTime      *time.Time
	Amount       *float64 // optional; when set, host should match payment amount
	NotifyData   map[string]any
}

// ApplyPaidResultStatusGate returns skip=true when payment is already paid (idempotent no-op).
func ApplyPaidResultStatusGate(status string) (skip bool, err error) {
	if status == StatusPaid {
		return true, nil
	}
	if status != StatusPending {
		return false, fmt.Errorf("%w: only pending payments can be marked paid", ErrStatusInvalid)
	}
	return false, nil
}
