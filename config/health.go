package config

import (
	"github.com/goravel/framework/facades"
)

func init() {
	config := facades.Config()
	config.Add("health", map[string]any{
		// Webhook URL for readiness failure alerts (empty = disabled)
		"ready_alert_webhook_url": config.Env("READY_ALERT_WEBHOOK_URL", ""),
	})
}
