package gateways

import (
	"strings"

	"goravel-payment"
)

// orderSubject is the order title shown to the payer (alipay subject / wechat description).
// Both providers reject an empty one, so fall back to the payment number.
func orderSubject(order *payment.Order) string {
	if s := strings.TrimSpace(order.Remark); s != "" {
		return s
	}
	return "Payment " + order.PaymentNo
}
