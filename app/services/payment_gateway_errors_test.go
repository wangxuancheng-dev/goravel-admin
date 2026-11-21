package services

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "goravel/app/errors"

	"goravel-payment"
)

func TestMapGatewayDriverErrorStripsSentinelPrefix(t *testing.T) {
	err := mapGatewayDriverError(fmt.Errorf("%w: out_trade_no is required", payment.ErrNotifyInvalid))
	be, ok := apperrors.GetBusinessError(err)
	require.True(t, ok)
	assert.Equal(t, apperrors.ErrPaymentNotifyInvalid.Code, be.Code)
	assert.Equal(t, "out_trade_no is required", be.Message)

	_, gateErr := payment.ApplyPaidResultStatusGate(payment.StatusFailed)
	be, ok = apperrors.GetBusinessError(mapGatewayStatusGateError(gateErr))
	require.True(t, ok)
	assert.Equal(t, apperrors.ErrPaymentStatusInvalid.Code, be.Code)
	assert.Equal(t, "only pending payments can be marked paid", be.Message)
}

func TestMapGatewayDriverErrorBareSentinelKeepsHostDefault(t *testing.T) {
	be, ok := apperrors.GetBusinessError(mapGatewayDriverError(payment.ErrNotImplemented))
	require.True(t, ok)
	assert.Equal(t, apperrors.ErrPaymentGatewayNotImplemented.Message, be.Message)
}

func TestMapGatewayDriverErrorPassesThrough(t *testing.T) {
	host := apperrors.ErrPaymentNotFound
	assert.Equal(t, host.Code, mustBusiness(t, mapGatewayDriverError(host)).Code)

	plain := errors.New("db down")
	assert.Same(t, plain, mapGatewayDriverError(plain))
}

func mustBusiness(t *testing.T, err error) *apperrors.BusinessError {
	t.Helper()
	be, ok := apperrors.GetBusinessError(err)
	require.True(t, ok)
	return be
}
