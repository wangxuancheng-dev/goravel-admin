package gateways

import (
	"context"
	"fmt"

	"github.com/go-pay/gopay"
	"github.com/go-pay/gopay/alipay"

	"goravel-payment"
)

func init() {
	payment.RegisterGateway(&alipayDriver{})
}

// alipayDriver: external Go module example (github.com/go-pay/gopay).
type alipayDriver struct{}

func (d *alipayDriver) Type() string { return "alipay" }

func (d *alipayDriver) Create(ctx context.Context, order *payment.Order, config map[string]any, _ string) (map[string]any, error) {
	if order == nil {
		return nil, fmt.Errorf("%w: order is nil", payment.ErrCreateFailed)
	}
	if config == nil {
		config = map[string]any{}
	}
	appID, _ := config["app_id"].(string)
	privateKey, _ := config["private_key"].(string)

	if appID == "" || privateKey == "" {
		return nil, fmt.Errorf("%w: alipay config incomplete", payment.ErrConfigRequired)
	}

	client, err := alipay.NewClient(appID, privateKey, false)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", payment.ErrCreateFailed, err)
	}

	notifyURL, _ := config["notify_url"].(string)
	if notifyURL == "" {
		notifyURL = payment.DefaultNotifyURL(ctx, "alipay")
	}
	client.SetNotifyUrl(notifyURL)

	bm := make(gopay.BodyMap)
	bm.Set("out_trade_no", order.PaymentNo)
	bm.Set("subject", order.Remark)
	bm.Set("total_amount", fmt.Sprintf("%.2f", order.Amount))
	bm.Set("product_code", "QUICK_MSECURITY_PAY")

	payURL, err := client.TradeAppPay(ctx, bm)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", payment.ErrCreateFailed, err)
	}
	return map[string]any{
		"payment_no": order.PaymentNo,
		"pay_url":    payURL,
	}, nil
}

func (d *alipayDriver) Query(_ context.Context, _ *payment.Order, _ map[string]any) (map[string]any, error) {
	return nil, payment.ErrNotImplemented
}

func (d *alipayDriver) Notify(_ context.Context, _ map[string]any, _ map[string]any) (*payment.PaidResult, error) {
	return nil, payment.ErrNotImplemented
}
