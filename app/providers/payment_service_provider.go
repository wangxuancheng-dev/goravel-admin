package providers

import (
	"context"
	"strings"

	"github.com/goravel/framework/contracts/foundation"
	"github.com/goravel/framework/facades"

	"goravel/app/services"
	"goravel/app/tenancy"
	"goravel/app/tenancyctx"

	"goravel-payment"
	_ "goravel-payment/gateways" // register mock / wechat / alipay via init()
)

// PaymentServiceProvider wires the reusable goravel-payment module for this app.
// Domain: goravel-payment (+ gateways). Orchestration (ApplyPaidResult) stays in services.
type PaymentServiceProvider struct{}

func (r *PaymentServiceProvider) Register(app foundation.Application) {}

func (r *PaymentServiceProvider) Boot(app foundation.Application) {
	payment.ResolvePaymentAmount = func(ctx context.Context, paymentNo string) (float64, error) {
		pay, err := services.NewPaymentService(ctx).GetPaymentByPaymentNo(paymentNo)
		if err != nil {
			return 0, err
		}
		return pay.Amount, nil
	}
	payment.ResolveNotifyURL = func(ctx context.Context, gatewayType string) string {
		base := strings.TrimRight(facades.Config().GetString("app.url"), "/")
		code := ""
		if ctx != nil {
			code, _ = tenancyctx.CodeFrom(ctx)
		}
		return base + tenancy.PaymentNotifyPath(code, gatewayType)
	}
}
