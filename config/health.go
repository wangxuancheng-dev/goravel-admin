package config

import (
	"github.com/goravel/framework/facades"
)

func init() {
	config := facades.Config()
	config.Add("health", map[string]any{
		// Webhook URL for readiness failure alerts (empty = disabled)
		"ready_alert_webhook_url": config.Env("READY_ALERT_WEBHOOK_URL", ""),
		// Queue backlog alert; empty falls back to ready_alert_webhook_url at runtime
		"queue_alert_webhook_url": config.Env("QUEUE_ALERT_WEBHOOK_URL", ""),
		// Pending job count threshold for queue:alert-backlog (default 100)
		"queue_alert_backlog_threshold": config.Env("QUEUE_ALERT_BACKLOG_THRESHOLD", 100),
	})
}
