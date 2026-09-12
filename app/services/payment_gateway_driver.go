package services

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	apperrors "goravel/app/errors"
	"goravel/app/models"
)

// PaymentGatewayDriver is one payment channel (mock / wechat / stripe / …).
// Register via RegisterPaymentGateway in init(); business success always goes through ApplyPaidResult.
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
	d, ok := LookupPaymentGateway(typ)
	if !ok || d == nil {
		return nil, apperrors.ErrInvalidPaymentType.WithMessage(fmt.Sprintf("unsupported payment gateway type: %s", typ))
	}
	return d, nil
}
