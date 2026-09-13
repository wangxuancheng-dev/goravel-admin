package gateways

import (
	"context"
	"fmt"

	"github.com/go-pay/gopay"
	"github.com/go-pay/gopay/alipay"

	apperrors "goravel/app/errors"
	"goravel/app/models"
	"goravel/app/payment"
)

func init() {
	payment.RegisterGateway(&alipayDriver{})
}

// alipayDriver: external Go module example (github.com/go-pay/gopay).
type alipayDriver struct{}

func (d *alipayDriver) Type() string { return "alipay" }

func (d *alipayDriver) Create(ctx context.Context, pay *models.Payment, _ *models.PaymentMethod, config map[string]any, _ string) (map[string]any, error) {
	appID, _ := config["app_id"].(string)
	privateKey, _ := config["private_key"].(string)

	if appID == "" || privateKey == "" {
		return nil, apperrors.ErrPaymentConfigRequired.WithMessage("支付宝配置不完整")
	}

	client, err := alipay.NewClient(appID, privateKey, false)
	if err != nil {
		return nil, apperrors.ErrCreatePaymentFailed.WithError(err)
	}

	notifyURL, _ := config["notify_url"].(string)
	if notifyURL == "" {
		notifyURL = payment.DefaultNotifyURL(ctx, "alipay")
	}
	client.SetNotifyUrl(notifyURL)

	bm := make(gopay.BodyMap)
	bm.Set("out_trade_no", pay.PaymentNo)
	bm.Set("subject", pay.Remark)
	bm.Set("total_amount", fmt.Sprintf("%.2f", pay.Amount))
	bm.Set("product_code", "QUICK_MSECURITY_PAY")

	payURL, err := client.TradeAppPay(ctx, bm)
	if err != nil {
		return nil, apperrors.ErrCreatePaymentFailed.WithError(err)
	}
	return map[string]any{
		"payment_no": pay.PaymentNo,
		"pay_url":    payURL,
	}, nil
}

func (d *alipayDriver) Query(_ context.Context, _ *models.Payment, _ *models.PaymentMethod, _ map[string]any) (map[string]any, error) {
	return nil, apperrors.ErrPaymentGatewayNotImplemented
}

func (d *alipayDriver) Notify(_ context.Context, _ *models.PaymentMethod, _ map[string]any) (*payment.PaidResult, error) {
	// TODO: gopay verify → return &payment.PaidResult{...} (services applies)
	return nil, apperrors.ErrPaymentGatewayNotImplemented
}
