package config

import "github.com/goravel/framework/facades"

func init() {
	config := facades.Config()
	config.Add("metrics", map[string]any{
		// METRICS_ENABLED=true exposes GET /metrics for Prometheus scrape.
		"enabled": config.Env("METRICS_ENABLED", false),
		// Optional bearer token: Authorization: Bearer <token> or ?token=
		"token": config.Env("METRICS_TOKEN", ""),
	})

	config.Add("oidc", map[string]any{
		"enabled":       config.Env("OIDC_ENABLED", false),
		"issuer":        config.Env("OIDC_ISSUER", ""),
		"client_id":     config.Env("OIDC_CLIENT_ID", ""),
		"client_secret": config.Env("OIDC_CLIENT_SECRET", ""),
		// Absolute callback URL registered at IdP, e.g. https://api.example.com/api/admin/auth/oidc/callback
		"redirect_url": config.Env("OIDC_REDIRECT_URL", ""),
		// Frontend landing after login, e.g. https://admin.example.com/login/oidc-callback
		"frontend_redirect": config.Env("OIDC_FRONTEND_REDIRECT", ""),
		"scopes":            config.Env("OIDC_SCOPES", "openid profile email"),
		"button_label":      config.Env("OIDC_BUTTON_LABEL", "Enterprise SSO"),
		// When true, create disabled admin if email unknown (not recommended for production).
		"auto_provision": config.Env("OIDC_AUTO_PROVISION", false),
	})

	config.Add("audit", map[string]any{
		// Retention days for scheduled operation/login log archives (export then delete).
		"retention_days": config.Env("AUDIT_LOG_RETENTION_DAYS", 30),
		// Optional SIEM / webhook fan-out for login + operation audit events.
		"webhook_url": config.Env("AUDIT_WEBHOOK_URL", ""),
	})
}
