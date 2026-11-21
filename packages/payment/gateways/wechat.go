package gateways

import (
	"context"
	"fmt"
	"math"

	"github.com/go-pay/gopay"
	"github.com/go-pay/gopay/wechat/v3"

	"goravel-payment"
)

func init() {
	payment.RegisterGateway(&wechatDriver{})
}

// wechatDriver: external Go module example (github.com/go-pay/gopay).
// Query/notify return NotImplemented until verify is wired.
type wechatDriver struct{}

func (d *wechatDriver) Type() string { return "wechat" }

func (d *wechatDriver) Create(ctx context.Context, order *payment.Order, config map[string]any, _ string) (map[string]any, error) {
	if order == nil {
		return nil, fmt.Errorf("%w: order is nil", payment.ErrCreateFailed)
	}
	if config == nil {
		config = map[string]any{}
	}
	appID, _ := config["app_id"].(string)
	mchID, _ := config["mch_id"].(string)
	apiV3Key, _ := config["api_v3_key"].(string)
	certSerialNo, _ := config["cert_serial_no"].(string)
	privateKeyPath, _ := config["private_key_path"].(string)

	if appID == "" || mchID == "" || apiV3Key == "" {
		return nil, fmt.Errorf("%w: wechat config incomplete", payment.ErrConfigRequired)
	}

	client, err := wechat.NewClientV3(appID, mchID, apiV3Key, certSerialNo)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", payment.ErrCreateFailed, err)
	}
	if privateKeyPath != "" {
		// client.SetPrivateKey(...) — wire per gopay docs
		_ = privateKeyPath
	}

	notifyURL, _ := config["notify_url"].(string)
	if notifyURL == "" {
		notifyURL = payment.DefaultNotifyURL(ctx, "wechat")
	}

	bm := make(gopay.BodyMap)
	bm.Set("out_trade_no", order.PaymentNo)
	bm.Set("description", order.Remark)
	bm.Set("amount", map[string]any{
		"total":    int(math.Round(order.Amount * 100)),
		"currency": "CNY",
	})
	bm.Set("notify_url", notifyURL)
	bm.Set("payer", map[string]any{
		"openid": config["openid"],
	})

	wxRsp, err := client.V3TransactionJsapi(ctx, bm)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", payment.ErrCreateFailed, err)
	}
	if wxRsp.Code != wechat.Success {
		return nil, fmt.Errorf("%w: %s", payment.ErrCreateFailed, wxRsp.Error)
	}
	return map[string]any{
		"payment_no": order.PaymentNo,
		"prepay_id":  wxRsp.Response.PrepayId,
	}, nil
}

func (d *wechatDriver) Query(_ context.Context, _ *payment.Order, _ map[string]any) (map[string]any, error) {
	return nil, payment.ErrNotImplemented
}

func (d *wechatDriver) Notify(_ context.Context, _ map[string]any, _ map[string]any) (*payment.PaidResult, error) {
	return nil, payment.ErrNotImplemented
}
