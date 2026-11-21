package providers

import (
	"context"

	"github.com/goravel/framework/contracts/foundation"

	"goravel/app/services"

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
	payment.ResolveNotifyURL = services.PaymentNotifyURL
}
