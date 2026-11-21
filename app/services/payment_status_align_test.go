package services

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"goravel/app/models"

	"goravel-payment"
)

// Host payment status strings must stay aligned with goravel-payment constants.
func TestPaymentStatusConstantsAlignWithGatewayPackage(t *testing.T) {
	assert.Equal(t, models.PaymentStatusPending, payment.StatusPending)
	assert.Equal(t, models.PaymentStatusPaid, payment.StatusPaid)
	assert.Equal(t, models.PaymentStatusFailed, payment.StatusFailed)
	assert.Equal(t, models.PaymentStatusCancelled, payment.StatusCancelled)
}
