package gateways

import (
	"context"
	"math"

	"github.com/go-pay/gopay"
	"github.com/go-pay/gopay/wechat/v3"

	apperrors "goravel/app/errors"
	"goravel/app/models"
	"goravel/app/payment"
)

func init() {
	payment.RegisterGateway(&wechatDriver{})
}

// wechatDriver: external Go module example (github.com/go-pay/gopay).
// Query/notify return NotImplemented until verify is wired.
type wechatDriver struct{}

func (d *wechatDriver) Type() string { return "wechat" }

func (d *wechatDriver) Create(ctx context.Context, pay *models.Payment, _ *models.PaymentMethod, config map[string]any, _ string) (map[string]any, error) {
	appID, _ := config["app_id"].(string)
	mchID, _ := config["mch_id"].(string)
	apiV3Key, _ := config["api_v3_key"].(string)
	certSerialNo, _ := config["cert_serial_no"].(string)
	privateKeyPath, _ := config["private_key_path"].(string)

	if appID == "" || mchID == "" || apiV3Key == "" {
		return nil, apperrors.ErrPaymentConfigRequired.WithMessage("微信支付配置不完整")
	}

	client, err := wechat.NewClientV3(appID, mchID, apiV3Key, certSerialNo)
	if err != nil {
		return nil, apperrors.ErrCreatePaymentFailed.WithError(err)
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
	bm.Set("out_trade_no", pay.PaymentNo)
	bm.Set("description", pay.Remark)
	bm.Set("amount", map[string]any{
		"total":    int(math.Round(pay.Amount * 100)),
		"currency": "CNY",
	})
	bm.Set("notify_url", notifyURL)
	bm.Set("payer", map[string]any{
		"openid": config["openid"],
	})

	wxRsp, err := client.V3TransactionJsapi(ctx, bm)
	if err != nil {
		return nil, apperrors.ErrCreatePaymentFailed.WithError(err)
	}
	if wxRsp.Code != wechat.Success {
		return nil, apperrors.ErrCreatePaymentFailed.WithMessage(wxRsp.Error)
	}
	return map[string]any{
		"payment_no": pay.PaymentNo,
		"prepay_id":  wxRsp.Response.PrepayId,
	}, nil
}

func (d *wechatDriver) Query(_ context.Context, _ *models.Payment, _ *models.PaymentMethod, _ map[string]any) (map[string]any, error) {
	return nil, apperrors.ErrPaymentGatewayNotImplemented
}

func (d *wechatDriver) Notify(_ context.Context, _ *models.PaymentMethod, _ map[string]any) (*payment.PaidResult, error) {
	// TODO: gopay verify → return &payment.PaidResult{...} (services applies)
	return nil, apperrors.ErrPaymentGatewayNotImplemented
}
