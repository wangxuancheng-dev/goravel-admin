package config

import (
	"github.com/beratkara/goralens"
	"github.com/goravel/framework/facades"
)

func init() {
	config := facades.Config()
	config.Add("logging", map[string]any{
		// Default Log Channel
		//
		// This option defines the default log channel that gets used when writing
		// messages to the logs. The name specified in this option should match
		// one of the channels defined in the "channels" configuration array.
		"default": config.Env("LOG_CHANNEL", "stack"),
		// Context keys excluded from log output (framework keys are excluded by default).
		"context": map[string]any{
			"exclude": []string{
				"admin",
				"user",
				"token",
				"permission_slug",
				"trace.traceContextKeyType",
				"go.opentelemetry.io/otel/trace.traceContextKeyType",
			},
		},
		// Log Channels
		//
		// Here you may configure the log channels for your application.
		// Available Drivers: "single", "daily", "custom", "stack"
		// Available Level: "debug", "info", "warning", "error", "fatal", "panic"
		"channels": map[string]any{
			"stack": map[string]any{
				"driver":   "stack",
				"channels": []string{"daily", "goralens"},
			},
			"single": map[string]any{
				"driver": "single",
				"path":   "storage/logs/goravel.log",
				"level":  config.Env("LOG_LEVEL", "debug"),
				"print":  false,
			},
			"daily": map[string]any{
				"driver": "daily",
				"path":   "storage/logs/goravel.log",
				"level":  config.Env("LOG_LEVEL", "debug"),
				"days":   7,
				"print":  true,
			},
			"otel": map[string]any{
				"driver":          "otel",
				"instrument_name": config.GetString("APP_NAME", "goravel/log"),
			},
			// Feeds the Goralens dashboard; a no-op while GORALENS_ENABLED is false.
			"goralens": map[string]any{
				"driver": "custom",
				"via":    goralens.LogDriver(),
			},
		},
	})
}
