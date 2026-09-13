package services

import (
	"fmt"
	"strings"

	"github.com/goravel/framework/facades"

	apperrors "goravel/app/errors"
	apppayment "goravel/app/payment"
)

// PaymentGatewayDriver is an alias of payment.GatewayDriver used by this package's allowlist helpers.
type PaymentGatewayDriver = apppayment.GatewayDriver

// RegisterPaymentGateway registers a driver (thin facade over payment.RegisterGateway).
func RegisterPaymentGateway(driver PaymentGatewayDriver) {
	apppayment.RegisterGateway(driver)
}

// LookupPaymentGateway returns a registered driver for type (e.g. "mock", "wechat").
func LookupPaymentGateway(typ string) (PaymentGatewayDriver, bool) {
	return apppayment.LookupGateway(typ)
}

// RegisteredPaymentGatewayTypes returns sorted registered type names.
func RegisteredPaymentGatewayTypes() []string {
	return apppayment.RegisteredGatewayTypes()
}

func requirePaymentGateway(typ string) (PaymentGatewayDriver, error) {
	typ = apppayment.NormalizeGatewayType(typ)
	d, ok := LookupPaymentGateway(typ)
	if !ok || d == nil {
		return nil, apperrors.ErrInvalidPaymentType.WithMessage(fmt.Sprintf("unsupported payment gateway type: %s", typ))
	}
	if !IsPaymentGatewayEnabled(typ) {
		return nil, apperrors.ErrPaymentGatewayDisabled.WithMessage(fmt.Sprintf("payment gateway disabled: %s", typ))
	}
	return d, nil
}

// IsPaymentGatewayEnabled reports whether type is allowed by PAYMENT_GATEWAYS_ENABLED
// and is registered. Empty allowlist = all registered drivers.
func IsPaymentGatewayEnabled(typ string) bool {
	typ = apppayment.NormalizeGatewayType(typ)
	if typ == "" {
		return false
	}
	if _, ok := LookupPaymentGateway(typ); !ok {
		return false
	}
	allow := paymentGatewayAllowlist()
	if len(allow) == 0 {
		return true
	}
	_, ok := allow[typ]
	return ok
}

// EnabledPaymentGateways returns registered drivers filtered by PAYMENT_GATEWAYS_ENABLED.
func EnabledPaymentGateways() []string {
	registered := RegisteredPaymentGatewayTypes()
	allow := paymentGatewayAllowlist()
	if len(allow) == 0 {
		return registered
	}
	out := make([]string, 0, len(registered))
	for _, typ := range registered {
		if allow[typ] {
			out = append(out, typ)
		}
	}
	return out
}

func paymentGatewayAllowlist() map[string]bool {
	raw := strings.TrimSpace(facades.Config().GetString("module.payment_gateways_enabled", ""))
	if raw == "" || raw == "*" || strings.EqualFold(raw, "all") {
		return nil
	}
	out := map[string]bool{}
	for _, part := range strings.Split(raw, ",") {
		part = apppayment.NormalizeGatewayType(part)
		if part != "" {
			out[part] = true
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
