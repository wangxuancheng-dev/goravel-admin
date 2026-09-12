package services

import (
	"context"
	"testing"

	"github.com/goravel/framework/facades"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "goravel/app/errors"
	"goravel/app/models"
	"goravel/app/tenancyctx"
)

func TestPaymentGatewayNotifyAndQueryStubs(t *testing.T) {
	svc := NewPaymentGatewayService(context.Background())

	_, err := svc.HandlePaymentNotify(&models.PaymentMethod{Type: "wechat", IsActive: true, Config: "{}"}, map[string]any{"out_trade_no": "x"})
	require.Error(t, err)
	be, ok := apperrors.GetBusinessError(err)
	require.True(t, ok)
	assert.Equal(t, apperrors.ErrPaymentGatewayNotImplemented.Code, be.Code)

	_, err = svc.HandlePaymentNotify(&models.PaymentMethod{Type: "alipay", IsActive: true, Config: "{}"}, map[string]any{})
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

func TestDefaultPaymentNotifyURLIncludesTenant(t *testing.T) {
	prevDriver := facades.Config().GetString("tenancy.driver")
	prevURL := facades.Config().GetString("app.url")
	facades.Config().Add("tenancy.driver", "database")
	facades.Config().Add("app.url", "https://example.com")
	t.Cleanup(func() {
		facades.Config().Add("tenancy.driver", prevDriver)
		facades.Config().Add("app.url", prevURL)
	})

	ctx := tenancyctx.WithTenant(context.Background(), 1, "tenant_1", "acme")
	assert.Equal(t, "https://example.com/api/payment/notify/wechat/acme", defaultPaymentNotifyURL(ctx, "wechat"))
	assert.Equal(t, "https://example.com/api/payment/notify/mock/acme", defaultPaymentNotifyURL(ctx, "mock"))
}
