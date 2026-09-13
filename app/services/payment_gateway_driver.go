package services

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/goravel/framework/facades"

	apperrors "goravel/app/errors"
	"goravel/app/models"
)

// PaymentGatewayDriver is one payment channel (mock / wechat / stripe / …).
//
// Multi-gateway scale path (do not invent new app/<vendor> domain packages):
//   - One file per channel: app/services/payment_gateway_<type>.go
//   - Register in init() via RegisterPaymentGateway; Type() must match payment_methods.type
//   - Notify / successful Query: verify → PaidResult → ApplyPaidResult only (never mutate order/payment yourself)
//   - Routes stay POST /api/payment/notify/{type}[/{tenant}]; no new notify controllers
//   - Implementation style (both first-class):
//       * External Go module / SDK: require in go.mod and call from the driver (see wechat/alipay + gopay)
//       * Hand-written from vendor docs: net/http + crypto/sign in the same driver file (see mock)
//   - Gate with PAYMENT_GATEWAYS_ENABLED; when payment_gateway_*.go grows past ~8–10 files,
//     drivers may move to app/payment/gateways/ while this registry + ApplyPaidResult stay in services
// Docs: website/docs/advanced/payments.md
type PaymentGatewayDriver interface {
	Type() string
	Create(ctx context.Context, payment *models.Payment, method *models.PaymentMethod, config map[string]any, clientIP string) (map[string]any, error)
	Query(ctx context.Context, payment *models.Payment, method *models.PaymentMethod, config map[string]any) (map[string]any, error)
	Notify(ctx context.Context, method *models.PaymentMethod, notifyData map[string]any) (*models.Payment, error)
}

var (
	paymentGatewayMu      sync.RWMutex
	paymentGatewayDrivers = map[string]PaymentGatewayDriver{}
)

// RegisterPaymentGateway registers a driver by Type() (lowercase). Safe to call from init().
func RegisterPaymentGateway(driver PaymentGatewayDriver) {
	if driver == nil {
		return
	}
	typ := normalizePaymentGatewayType(driver.Type())
	if typ == "" {
		return
	}
	paymentGatewayMu.Lock()
	defer paymentGatewayMu.Unlock()
	paymentGatewayDrivers[typ] = driver
}

// LookupPaymentGateway returns a registered driver for type (e.g. "mock", "wechat").
func LookupPaymentGateway(typ string) (PaymentGatewayDriver, bool) {
	paymentGatewayMu.RLock()
	defer paymentGatewayMu.RUnlock()
	d, ok := paymentGatewayDrivers[normalizePaymentGatewayType(typ)]
	return d, ok
}

// RegisteredPaymentGatewayTypes returns sorted registered type names.
func RegisteredPaymentGatewayTypes() []string {
	paymentGatewayMu.RLock()
	defer paymentGatewayMu.RUnlock()
	out := make([]string, 0, len(paymentGatewayDrivers))
	for k := range paymentGatewayDrivers {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func normalizePaymentGatewayType(typ string) string {
	return strings.ToLower(strings.TrimSpace(typ))
}

func requirePaymentGateway(typ string) (PaymentGatewayDriver, error) {
	typ = normalizePaymentGatewayType(typ)
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
	typ = normalizePaymentGatewayType(typ)
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
		part = normalizePaymentGatewayType(part)
		if part != "" {
			out[part] = true
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
