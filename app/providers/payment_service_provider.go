package providers

import (
	"context"

	"github.com/goravel/framework/contracts/foundation"

	apppayment "goravel/app/payment"
	_ "goravel/app/payment/gateways" // register mock / wechat / alipay via init()
	"goravel/app/services"
)

// PaymentServiceProvider wires the payment domain package for this app.
// Domain: app/payment (+ gateways). Orchestration (ApplyPaidResult) stays in services.
// To extract later: move payment/ + gateways/ to a module; keep this provider (or package setup)
// as the only place that blank-imports drivers and injects ResolvePaymentAmount.
type PaymentServiceProvider struct{}

func (r *PaymentServiceProvider) Register(app foundation.Application) {}

func (r *PaymentServiceProvider) Boot(app foundation.Application) {
	// Hand-written drivers (e.g. mock HMAC) resolve amounts without importing services.
	apppayment.ResolvePaymentAmount = func(ctx context.Context, paymentNo string) (float64, error) {
		pay, err := services.NewPaymentService(ctx).GetPaymentByPaymentNo(paymentNo)
		if err != nil {
			return 0, err
		}
		return pay.Amount, nil
	}
}
