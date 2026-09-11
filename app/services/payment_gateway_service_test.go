package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "goravel/app/errors"
	"goravel/app/models"
)

func TestPaymentGatewayNotifyAndQueryStubs(t *testing.T) {
	svc := NewPaymentGatewayService(context.Background())

	_, err := svc.HandlePaymentNotify(&models.PaymentMethod{Type: "wechat", IsActive: true}, map[string]any{"out_trade_no": "x"})
	require.Error(t, err)
	be, ok := apperrors.GetBusinessError(err)
	require.True(t, ok)
	assert.Equal(t, apperrors.ErrPaymentGatewayNotImplemented.Code, be.Code)

	_, err = svc.HandlePaymentNotify(&models.PaymentMethod{Type: "alipay", IsActive: true}, map[string]any{})
	require.Error(t, err)
	be, ok = apperrors.GetBusinessError(err)
	require.True(t, ok)
	assert.Equal(t, apperrors.ErrPaymentGatewayNotImplemented.Code, be.Code)

	_, err = svc.HandlePaymentNotify(&models.PaymentMethod{Type: "unknown", IsActive: true}, nil)
	require.Error(t, err)
	be, ok = apperrors.GetBusinessError(err)
	require.True(t, ok)
	assert.Equal(t, apperrors.ErrInvalidPaymentType.Code, be.Code)

	impl := svc.(*PaymentGatewayServiceImpl)
	_, err = impl.queryWechatPayment(&models.Payment{PaymentNo: "P1"}, map[string]any{})
	require.Error(t, err)
	be, ok = apperrors.GetBusinessError(err)
	require.True(t, ok)
	assert.Equal(t, apperrors.ErrPaymentGatewayNotImplemented.Code, be.Code)

	_, err = impl.queryAlipayPayment(&models.Payment{PaymentNo: "P1"}, map[string]any{})
	require.Error(t, err)
	be, ok = apperrors.GetBusinessError(err)
	require.True(t, ok)
	assert.Equal(t, apperrors.ErrPaymentGatewayNotImplemented.Code, be.Code)
}
