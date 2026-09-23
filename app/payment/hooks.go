package payment

import "context"

// ResolvePaymentAmount looks up a payment's expected amount by payment_no.
// Wired by providers.PaymentServiceProvider so gateway drivers never import app/services.
var ResolvePaymentAmount func(ctx context.Context, paymentNo string) (float64, error)
