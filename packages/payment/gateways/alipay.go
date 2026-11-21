package gateways

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-pay/gopay"
	"github.com/go-pay/gopay/alipay"

	"goravel-payment"
)

func init() {
	payment.RegisterGateway(&alipayDriver{})
}

// alipayDriver: external Go module example (github.com/go-pay/gopay), App pay.
//
// Config: app_id, private_key (app private key), sandbox ("true"/"1" = openapi sandbox;
// default production), notify_url (optional; else payment.ResolveNotifyURL).
// Create returns order_string: the signed string the Alipay App SDK takes, not a URL.
type alipayDriver struct{}

func (d *alipayDriver) Type() string { return "alipay" }

func (d *alipayDriver) Create(ctx context.Context, order *payment.Order, config map[string]any, _ string) (map[string]any, error) {
	if order == nil {
		return nil, fmt.Errorf("%w: order is nil", payment.ErrCreateFailed)
	}
	appID := strings.TrimSpace(payment.StringFromConfig(config, "app_id"))
	privateKey := alipayKeyBody(payment.StringFromConfig(config, "private_key"))
	if appID == "" || privateKey == "" {
		return nil, fmt.Errorf("%w: alipay needs app_id and private_key", payment.ErrConfigRequired)
	}
	notifyURL, err := payment.RequireNotifyURL(ctx, "alipay", config)
	if err != nil {
		return nil, err
	}

	sandbox := payment.BoolFromConfig(config, "sandbox")
	client, err := alipay.NewClient(appID, privateKey, !sandbox)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", payment.ErrCreateFailed, err)
	}
	client.SetNotifyUrl(notifyURL)

	bm := make(gopay.BodyMap)
	bm.Set("out_trade_no", order.PaymentNo)
	bm.Set("subject", orderSubject(order))
	bm.Set("total_amount", fmt.Sprintf("%.2f", order.Amount))
	bm.Set("product_code", "QUICK_MSECURITY_PAY")

	orderString, err := client.TradeAppPay(ctx, bm)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", payment.ErrCreateFailed, err)
	}
	return map[string]any{
		"payment_no":   order.PaymentNo,
		"order_string": orderString,
		"sandbox":      sandbox,
	}, nil
}

// alipayKeyBody returns the bare base64 key gopay's alipay.NewClient expects: it adds
// its own PEM header, so a pasted full PEM (-----BEGIN ...----- lines) must be unwrapped.
func alipayKeyBody(raw string) string {
	var b strings.Builder
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "-----") {
			continue
		}
		b.WriteString(line)
	}
	return b.String()
}

func (d *alipayDriver) Query(_ context.Context, _ *payment.Order, _ map[string]any) (map[string]any, error) {
	return nil, payment.ErrNotImplemented
}

func (d *alipayDriver) Notify(_ context.Context, _ map[string]any, _ map[string]any) (*payment.PaidResult, error) {
	return nil, payment.ErrNotImplemented
}
