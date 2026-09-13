package services

import (
	"context"

	apppayment "goravel/app/payment"
	_ "goravel/app/payment/gateways" // register mock / wechat / alipay drivers
)

func init() {
	// Let hand-written drivers (e.g. mock HMAC) resolve amounts without importing services.
	apppayment.ResolvePaymentAmount = func(ctx context.Context, paymentNo string) (float64, error) {
		pay, err := NewPaymentService(ctx).GetPaymentByPaymentNo(paymentNo)
		if err != nil {
			return 0, err
		}
		return pay.Amount, nil
	}
}
