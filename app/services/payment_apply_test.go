package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "goravel/app/errors"
	"goravel/app/models"
)

func TestMockNotifySignRoundTrip(t *testing.T) {
	sign := mockNotifySign("PAY1", "SUCCESS", 9.9, "secret")
	assert.True(t, hmacEqual(sign, mockNotifySign("PAY1", "SUCCESS", 9.9, "secret")))
	assert.False(t, hmacEqual(sign, mockNotifySign("PAY1", "SUCCESS", 9.91, "secret")))
}

func TestMockNotifyRequiresOutTradeNo(t *testing.T) {
	svc := NewPaymentGatewayService(context.Background()).(*PaymentGatewayServiceImpl)
	_, err := svc.handleMockNotify(&models.PaymentMethod{Type: "mock", Config: "{}"}, map[string]any{"trade_status": "SUCCESS"})
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
