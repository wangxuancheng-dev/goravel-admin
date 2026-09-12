package utils

import (
	"testing"

	"github.com/goravel/framework/facades"
)

func TestOrdersEnabledRespectsModuleConfig(t *testing.T) {
	prevOrders := facades.Config().GetBool("module.orders_enabled", true)
	prevPayments := facades.Config().GetBool("module.payments_enabled", false)
	prevGateways := facades.Config().GetString("module.payment_gateways_enabled", "")
	prevFrontend := facades.Config().GetString("module.code_generator_frontend", "vue,react")
	t.Cleanup(func() {
		facades.Config().Add("module", map[string]any{
			"orders_enabled":           prevOrders,
			"payments_enabled":         prevPayments,
			"payment_gateways_enabled": prevGateways,
			"code_generator_frontend":  prevFrontend,
		})
	})

	facades.Config().Add("module", map[string]any{
		"orders_enabled":           false,
		"payments_enabled":         prevPayments,
		"payment_gateways_enabled": prevGateways,
		"code_generator_frontend":  prevFrontend,
	})
	if OrdersEnabled() {
		t.Fatal("expected OrdersEnabled=false")
	}

	facades.Config().Add("module", map[string]any{
		"orders_enabled":           true,
		"payments_enabled":         prevPayments,
		"payment_gateways_enabled": prevGateways,
		"code_generator_frontend":  prevFrontend,
	})
	if !OrdersEnabled() {
		t.Fatal("expected OrdersEnabled=true")
	}
}
