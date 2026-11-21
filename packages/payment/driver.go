package payment

import (
	"context"
	"sort"
	"strings"
	"sync"
)

// GatewayDriver is one payment channel (mock / wechat / stripe / ...).
//
// Host boundary:
//   - Registry + PaidResult + helpers live here; drivers in gateways/ (one file per type).
//   - Drivers must NOT import host services. Amount lookup uses ResolvePaymentAmount.
//   - Notify only returns *PaidResult; the host owns DB writes.
//   - Host wires drivers (blank-import gateways) and hooks via a ServiceProvider.
//
// Implementation style (both first-class):
//   - External Go module / SDK (see gateways/wechat.go)
//   - Hand-written from vendor docs (see gateways/mock.go)
type GatewayDriver interface {
	Type() string
	Create(ctx context.Context, order *Order, config map[string]any, clientIP string) (map[string]any, error)
	Query(ctx context.Context, order *Order, config map[string]any) (map[string]any, error)
	// Notify verifies the provider callback and returns a PaidResult.
	// config is the parsed payment-method JSON; do not write payment/order rows here.
	Notify(ctx context.Context, config map[string]any, notifyData map[string]any) (*PaidResult, error)
}

var (
	gatewayMu      sync.RWMutex
	gatewayDrivers = map[string]GatewayDriver{}
)

// RegisterGateway registers a driver by Type() (lowercase). Safe to call from init().
func RegisterGateway(driver GatewayDriver) {
	if driver == nil {
		return
	}
	typ := NormalizeGatewayType(driver.Type())
	if typ == "" {
		return
	}
	gatewayMu.Lock()
	defer gatewayMu.Unlock()
	gatewayDrivers[typ] = driver
}

// LookupGateway returns a registered driver for type (e.g. "mock", "wechat").
func LookupGateway(typ string) (GatewayDriver, bool) {
	gatewayMu.RLock()
	defer gatewayMu.RUnlock()
	d, ok := gatewayDrivers[NormalizeGatewayType(typ)]
	return d, ok
}

// RegisteredGatewayTypes returns sorted registered type names.
func RegisteredGatewayTypes() []string {
	gatewayMu.RLock()
	defer gatewayMu.RUnlock()
	out := make([]string, 0, len(gatewayDrivers))
	for k := range gatewayDrivers {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// NormalizeGatewayType lowercases and trims a gateway type string.
func NormalizeGatewayType(typ string) string {
	return strings.ToLower(strings.TrimSpace(typ))
}

// UnregisterGateway removes a driver (tests only).
func UnregisterGateway(typ string) {
	typ = NormalizeGatewayType(typ)
	if typ == "" {
		return
	}
	gatewayMu.Lock()
	defer gatewayMu.Unlock()
	delete(gatewayDrivers, typ)
}
