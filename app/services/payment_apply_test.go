package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "goravel/app/errors"
	"goravel/app/models"
	apppayment "goravel/app/payment"
)

func TestMockNotifyRequiresOutTradeNo(t *testing.T) {
	d, ok := LookupPaymentGateway("mock")
	require.True(t, ok)
	_, err := d.Notify(context.Background(), &models.PaymentMethod{Type: "mock", Config: "{}"}, map[string]any{"trade_status": "SUCCESS"})
	require.Error(t, err)
	be, ok := apperrors.GetBusinessError(err)
	require.True(t, ok)
	assert.Equal(t, apperrors.ErrPaymentNotifyInvalid.Code, be.Code)
}

func TestPaidResultRequiresPaymentNo(t *testing.T) {
	_, err := ApplyPaidResult(context.Background(), PaidResult{})
	require.Error(t, err)
	be, ok := apperrors.GetBusinessError(err)
	require.True(t, ok)
	assert.Equal(t, apperrors.ErrPaymentNoRequired.Code, be.Code)
}

func TestApplyPaidResultDecisionIdempotent(t *testing.T) {
	skip, err := apppayment.ApplyPaidResultStatusGate(models.PaymentStatusPaid)
	require.NoError(t, err)
	assert.True(t, skip, "already-paid payments must short-circuit without order writes")

	skip, err = apppayment.ApplyPaidResultStatusGate(models.PaymentStatusPending)
	require.NoError(t, err)
	assert.False(t, skip)

	skip, err = apppayment.ApplyPaidResultStatusGate(models.PaymentStatusFailed)
	require.Error(t, err)
	assert.False(t, skip)
}
