package services

import (
	"fmt"
	"strings"

	"github.com/goravel/framework/facades"

	apperrors "goravel/app/errors"

	"goravel-payment"
)

// requirePaymentGateway returns a registered driver that is also allowlisted.
// Registry API lives in goravel-payment (RegisterGateway / LookupGateway).
func requirePaymentGateway(typ string) (payment.GatewayDriver, error) {
	typ = payment.NormalizeGatewayType(typ)
	d, ok := payment.LookupGateway(typ)
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
	typ = payment.NormalizeGatewayType(typ)
	if typ == "" {
		return false
	}
	if _, ok := payment.LookupGateway(typ); !ok {
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
	registered := payment.RegisteredGatewayTypes()
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
		part = payment.NormalizeGatewayType(part)
		if part != "" {
			out[part] = true
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
