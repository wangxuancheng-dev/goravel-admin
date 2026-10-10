package config

import (
	"github.com/goravel/framework/facades"
)

func init() {
	config := facades.Config()
	config.Add("module", map[string]any{
		// Comma-separated payment gateway types. Empty = all registered drivers.
		"payment_gateways_enabled": config.Env("PAYMENT_GATEWAYS_ENABLED", ""),
		// Code generator frontends: react / vue / react,vue
		"code_generator_frontend": config.Env("CODE_GENERATOR_FRONTEND", "react,vue"),
	})
}
