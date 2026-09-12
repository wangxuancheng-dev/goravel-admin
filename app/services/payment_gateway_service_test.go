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

	wechat, ok := LookupPaymentGateway("wechat")
	require.True(t, ok)
	_, err = wechat.Query(context.Background(), &models.Payment{PaymentNo: "P1"}, &models.PaymentMethod{Type: "wechat"}, map[string]any{})
	require.Error(t, err)
	be, ok = apperrors.GetBusinessError(err)
	require.True(t, ok)
	assert.Equal(t, apperrors.ErrPaymentGatewayNotImplemented.Code, be.Code)

	alipay, ok := LookupPaymentGateway("alipay")
	require.True(t, ok)
	_, err = alipay.Query(context.Background(), &models.Payment{PaymentNo: "P1"}, &models.PaymentMethod{Type: "alipay"}, map[string]any{})
	require.Error(t, err)
	be, ok = apperrors.GetBusinessError(err)
	require.True(t, ok)
	assert.Equal(t, apperrors.ErrPaymentGatewayNotImplemented.Code, be.Code)
}

func TestRegisteredPaymentGatewaysIncludeBuiltin(t *testing.T) {
	types := RegisteredPaymentGatewayTypes()
	assert.Contains(t, types, "mock")
	assert.Contains(t, types, "wechat")
	assert.Contains(t, types, "alipay")
}

func TestPaymentGatewayAllowlist(t *testing.T) {
	prevOrders := facades.Config().GetBool("module.orders_enabled", true)
	prevPayments := facades.Config().GetBool("module.payments_enabled", false)
	prevGateways := facades.Config().GetString("module.payment_gateways_enabled", "")
	prevFrontend := facades.Config().GetString("module.code_generator_frontend", "vue,react")
	restore := func(gateways string) {
		facades.Config().Add("module", map[string]any{
			"orders_enabled":           prevOrders,
			"payments_enabled":         prevPayments,
			"payment_gateways_enabled": gateways,
			"code_generator_frontend":  prevFrontend,
		})
	}
	t.Cleanup(func() { restore(prevGateways) })

	restore("")
	assert.True(t, IsPaymentGatewayEnabled("mock"))
	assert.True(t, IsPaymentGatewayEnabled("wechat"))
	assert.ElementsMatch(t, RegisteredPaymentGatewayTypes(), EnabledPaymentGateways())

	restore("wechat,alipay")
	assert.False(t, IsPaymentGatewayEnabled("mock"))
	assert.True(t, IsPaymentGatewayEnabled("wechat"))
	assert.True(t, IsPaymentGatewayEnabled("alipay"))
	assert.Equal(t, []string{"alipay", "wechat"}, EnabledPaymentGateways())

	_, err := requirePaymentGateway("mock")
	require.Error(t, err)
	be, ok := apperrors.GetBusinessError(err)
	require.True(t, ok)
	assert.Equal(t, apperrors.ErrPaymentGatewayDisabled.Code, be.Code)
}

func TestRegisterPaymentGatewayCustomType(t *testing.T) {
	const typ = "demo_ext_channel"
	RegisterPaymentGateway(&stubPaymentDriver{typ: typ})
	t.Cleanup(func() {
		paymentGatewayMu.Lock()
		delete(paymentGatewayDrivers, typ)
		paymentGatewayMu.Unlock()
	})

	d, ok := LookupPaymentGateway(typ)
	require.True(t, ok)
	assert.Equal(t, typ, d.Type())

	_, err := NewPaymentGatewayService(context.Background()).HandlePaymentNotify(
		&models.PaymentMethod{Type: typ, Config: "{}"},
		map[string]any{},
	)
	require.Error(t, err)
	be, ok := apperrors.GetBusinessError(err)
	require.True(t, ok)
	assert.Equal(t, apperrors.ErrPaymentGatewayNotImplemented.Code, be.Code)
}

type stubPaymentDriver struct{ typ string }

func (d *stubPaymentDriver) Type() string { return d.typ }
func (d *stubPaymentDriver) Create(context.Context, *models.Payment, *models.PaymentMethod, map[string]any, string) (map[string]any, error) {
	return nil, apperrors.ErrPaymentGatewayNotImplemented
}
func (d *stubPaymentDriver) Query(context.Context, *models.Payment, *models.PaymentMethod, map[string]any) (map[string]any, error) {
	return nil, apperrors.ErrPaymentGatewayNotImplemented
}
func (d *stubPaymentDriver) Notify(context.Context, *models.PaymentMethod, map[string]any) (*models.Payment, error) {
	return nil, apperrors.ErrPaymentGatewayNotImplemented
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
