package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "goravel/app/errors"
	"goravel/app/models"

	"goravel-payment"
	_ "goravel-payment/gateways" // unit tests do not boot PaymentServiceProvider
)

func TestMockGatewayRegistered(t *testing.T) {
	d, ok := payment.LookupGateway("mock")
	require.True(t, ok)
	require.NotNil(t, d)
	assert.Equal(t, "mock", d.Type())
}

func TestApplyPaidResultRequiresPaymentNo(t *testing.T) {
	_, err := ApplyPaidResult(context.Background(), payment.PaidResult{})
	require.Error(t, err)
	be, ok := apperrors.GetBusinessError(err)
	require.True(t, ok)
	assert.Equal(t, apperrors.ErrPaymentNoRequired.Code, be.Code)
}

func TestApplyPaidResultDecisionIdempotent(t *testing.T) {
	skip, err := payment.ApplyPaidResultStatusGate(models.PaymentStatusPaid)
	require.NoError(t, err)
	assert.True(t, skip)

	skip, err = payment.ApplyPaidResultStatusGate(models.PaymentStatusPending)
	require.NoError(t, err)
	assert.False(t, skip)

	skip, err = payment.ApplyPaidResultStatusGate(models.PaymentStatusFailed)
	require.Error(t, err)
	assert.False(t, skip)
	assert.True(t, errors.Is(err, payment.ErrStatusInvalid))
	mapped := mapGatewayStatusGateError(err)
	be, ok := apperrors.GetBusinessError(mapped)
	require.True(t, ok)
	assert.Equal(t, apperrors.ErrPaymentStatusInvalid.Code, be.Code)
}
