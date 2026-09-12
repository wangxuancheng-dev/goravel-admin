package config

import (
	"github.com/goravel/framework/facades"
)

func init() {
	config := facades.Config()
	config.Add("notification", map[string]any{
		"mail_enabled":    config.Env("NOTIFICATION_MAIL_ENABLED", false),
		"webhook_enabled": config.Env("NOTIFICATION_WEBHOOK_ENABLED", false),
		"webhook_url":     config.Env("NOTIFICATION_WEBHOOK_URL", ""),
	})
}
