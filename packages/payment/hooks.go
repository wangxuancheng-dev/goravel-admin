package payment

import "context"

// ResolvePaymentAmount looks up a payment's expected amount by payment_no.
// Host apps wire this so gateway drivers never import host services.
var ResolvePaymentAmount func(ctx context.Context, paymentNo string) (float64, error)

// ResolveNotifyURL builds the public notify URL for a gateway type.
// Host apps typically use APP_URL + /api/payment/notify/{type}[/{tenant}].
var ResolveNotifyURL func(ctx context.Context, gatewayType string) string
