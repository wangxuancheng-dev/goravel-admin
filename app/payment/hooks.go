package payment

import "context"

// ResolvePaymentAmount looks up a payment's expected amount by payment_no.
// Wired from services so gateway drivers (e.g. mock HMAC) never import services.
var ResolvePaymentAmount func(ctx context.Context, paymentNo string) (float64, error)
