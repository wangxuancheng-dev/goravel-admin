package services

import (
	"context"
	"strings"

	"github.com/goravel/framework/facades"

	"goravel/app/tenancy"
	"goravel/app/tenancyctx"
)

// PaymentNotifyURL builds APP_URL + /api/payment/notify/{type}[/{tenant}].
// Wired into goravel-payment as payment.ResolveNotifyURL by PaymentServiceProvider.
func PaymentNotifyURL(ctx context.Context, gatewayType string) string {
	base := strings.TrimRight(facades.Config().GetString("app.url"), "/")
	code := ""
	if ctx != nil {
		code, _ = tenancyctx.CodeFrom(ctx)
	}
	return base + tenancy.PaymentNotifyPath(code, gatewayType)
}
