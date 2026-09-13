package payment

import (
	"time"

	apperrors "goravel/app/errors"
	"goravel/app/models"
)

// PaidResult is the normalized outcome after a gateway verifies a successful payment.
// Gateways should only verify + map fields; business writes go through services.ApplyPaidResult
// (status gate lives here as ApplyPaidResultStatusGate).
type PaidResult struct {
	PaymentNo    string
	ThirdPartyNo string
	PayTime      *time.Time
	Amount       *float64 // optional; when set, must match payment.Amount
	NotifyData   map[string]any
}

// ApplyPaidResultStatusGate returns skip=true when payment is already paid (idempotent no-op).
func ApplyPaidResultStatusGate(status string) (skip bool, err error) {
	if status == models.PaymentStatusPaid {
		return true, nil
	}
	if status != models.PaymentStatusPending {
		return false, apperrors.ErrPaymentStatusInvalid.WithMessage("only pending payments can be marked paid")
	}
	return false, nil
}
