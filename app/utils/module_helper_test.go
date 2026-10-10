package utils

import (
	"testing"

	"github.com/goravel/framework/facades"
)

func TestDisabledModuleMenuSlugsOptionalTogglesOnly(t *testing.T) {
	prevGateways := facades.Config().GetString("module.payment_gateways_enabled", "")
	prevFrontend := facades.Config().GetString("module.code_generator_frontend", "react,vue")
	facades.Config().Add("module", map[string]any{
		"payment_gateways_enabled": prevGateways,
		"code_generator_frontend":  prevFrontend,
	})
	t.Cleanup(func() {
		facades.Config().Add("module", map[string]any{
			"payment_gateways_enabled": prevGateways,
			"code_generator_frontend":  prevFrontend,
		})
	})

	disabled := DisabledModuleMenuSlugs()
	for _, slug := range []string{"order", "payment", "payment-method", "demo-activity", "article"} {
		if disabled[slug] {
			t.Fatalf("unexpected disabled slug %q: %+v", slug, disabled)
		}
	}
}
