package services

import (
	"context"
	"fmt"

	"github.com/go-pay/gopay"
	"github.com/go-pay/gopay/alipay"

	apperrors "goravel/app/errors"
	"goravel/app/models"
)

func init() {
	RegisterPaymentGateway(&alipayPaymentDriver{})
}

// alipayPaymentDriver: example using an external Go module (github.com/go-pay/gopay).
// Same pattern as wechat — SDK in go.mod, thin adapter here. Query/notify NotImplemented until verify is wired.
type alipayPaymentDriver struct{}

func (d *alipayPaymentDriver) Type() string { return "alipay" }

func (d *alipayPaymentDriver) Create(ctx context.Context, payment *models.Payment, _ *models.PaymentMethod, config map[string]any, _ string) (map[string]any, error) {
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
		notifyURL = defaultPaymentNotifyURL(ctx, "alipay")
	}
	client.SetNotifyUrl(notifyURL)

	bm := make(gopay.BodyMap)
	bm.Set("out_trade_no", payment.PaymentNo)
	bm.Set("subject", payment.Remark)
	bm.Set("total_amount", fmt.Sprintf("%.2f", payment.Amount))
	bm.Set("product_code", "QUICK_MSECURITY_PAY")

	payURL, err := client.TradeAppPay(ctx, bm)
	if err != nil {
		return nil, apperrors.ErrCreatePaymentFailed.WithError(err)
	}
	return map[string]any{
		"payment_no": payment.PaymentNo,
		"pay_url":    payURL,
	}, nil
}

func (d *alipayPaymentDriver) Query(_ context.Context, _ *models.Payment, _ *models.PaymentMethod, _ map[string]any) (map[string]any, error) {
	return nil, apperrors.ErrPaymentGatewayNotImplemented
}

func (d *alipayPaymentDriver) Notify(_ context.Context, _ *models.PaymentMethod, _ map[string]any) (*models.Payment, error) {
	// TODO: gopay verify → PaidResult → return ApplyPaidResult(ctx, result)
	return nil, apperrors.ErrPaymentGatewayNotImplemented
}
