package config

import (
	"github.com/goravel/framework/facades"
)

func init() {
	config := facades.Config()
	config.Add("desensitize", map[string]any{
		// Process default. Per-tenant (or single-site) override: configs table
		// group=desensitize key=enabled (Admin → System → Config). Missing DB row
		// falls back to this env default — so only some tenants need to turn it on/off.
		"enabled": config.Env("DESENSITIZE_ENABLED", true),

		// Role slugs that see unmasked values (comma-separated).
		// DB override: configs.desensitize.bypass_role_slugs
		"bypass_role_slugs": config.Env("DESENSITIZE_BYPASS_ROLE_SLUGS", "super-admin"),

		// Modes that apply masking. Detail/show stays plain so edit forms do not save masks.
		// Supported: list, export
		"modes": []string{"list", "export"},

		// module -> field -> strategy (phone | email | hide | keep_ends)
		"modules": map[string]any{
			"user": map[string]any{
				"phone": "phone",
				"email": "email",
			},
			"admin": map[string]any{
				"phone": "phone",
				"email": "email",
			},
		},
	})
}
