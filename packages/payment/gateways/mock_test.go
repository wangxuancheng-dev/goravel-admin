package gateways

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"goravel-payment"
)

func TestMockNotifySignRoundTrip(t *testing.T) {
	sign := mockNotifySign("PAY1", "SUCCESS", 9.9, "secret")
	assert.True(t, hmacEqual(sign, mockNotifySign("PAY1", "SUCCESS", 9.9, "secret")))
	assert.False(t, hmacEqual(sign, mockNotifySign("PAY1", "SUCCESS", 9.91, "secret")))
}

func TestMockNotifyRequiresOutTradeNo(t *testing.T) {
	_, err := (&mockDriver{}).Notify(context.Background(), map[string]any{"shared_secret": "s"}, map[string]any{"trade_status": "SUCCESS"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, payment.ErrNotifyInvalid))
}

func TestMockNotifyRejectsBadSign(t *testing.T) {
	_, err := (&mockDriver{}).Notify(context.Background(), map[string]any{"shared_secret": "s"}, map[string]any{
		"out_trade_no": "PAY1",
		"trade_status": "SUCCESS",
		"amount":       9.9,
		"sign":         mockNotifySign("PAY1", "SUCCESS", 9.9, "other-secret"),
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, payment.ErrNotifyInvalid))
}

func TestMockCreateThenNotify(t *testing.T) {
	cfg := map[string]any{"shared_secret": "s"}
	d := &mockDriver{}
	out, err := d.Create(context.Background(), &payment.Order{PaymentNo: "PAY1", Amount: 9.9}, cfg, "")
	require.NoError(t, err)

	result, err := d.Notify(context.Background(), cfg, map[string]any{
		"out_trade_no": "PAY1",
		"trade_status": "SUCCESS",
		"amount":       9.9,
		"sign":         out["sign"],
	})
	require.NoError(t, err)
	assert.Equal(t, "PAY1", result.PaymentNo)
	require.NotNil(t, result.Amount)
	assert.InDelta(t, 9.9, *result.Amount, 1e-9)
}

func TestProviderGatewaysRequireNotifyURL(t *testing.T) {
	prev := payment.ResolveNotifyURL
	payment.ResolveNotifyURL = nil
	t.Cleanup(func() { payment.ResolveNotifyURL = prev })

	order := &payment.Order{PaymentNo: "PAY1", Amount: 1}
	_, err := (&wechatDriver{}).Create(context.Background(), order, map[string]any{
		"app_id": "a", "mch_id": "m", "api_v3_key": "k",
	}, "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, payment.ErrConfigRequired), "wechat: %v", err)
	_, err = (&alipayDriver{}).Create(context.Background(), order, map[string]any{
		"app_id": "a", "private_key": "k",
	}, "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, payment.ErrConfigRequired), "alipay: %v", err)
}
