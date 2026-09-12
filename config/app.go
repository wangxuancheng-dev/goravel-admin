package config

import (
	"slices"
	"strings"

	"github.com/goravel/framework/facades"
	"github.com/goravel/framework/support/carbon"
	"github.com/spf13/cast"

	"goravel/lang"
)

func parseDisabledRunners(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{}
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

// mergeDisabledRunners appends goravel:telemetry when OTEL exporters are not configured.
func mergeDisabledRunners(user []string, tracesExporter, metricsExporter string) []string {
	if tracesExporter != "" || metricsExporter != "" {
		return user
	}
	for _, pattern := range user {
		switch pattern {
		case "goravel:telemetry", "*", "goravel:*":
			return user
		}
	}
	return append(slices.Clone(user), "goravel:telemetry")
}

// Boot Start all init methods of the current folder to bootstrap all config.
func Boot() {}
func init() {
	config := facades.Config()
	config.Add("app", map[string]any{
		// Application Name
		//
		// This value is the name of your application. This value is used when the
		// framework needs to place the application's name in a notification or
		// any other location as required by the application or its packages.
		"name": config.Env("APP_NAME", "Goravel"),
		// Optional build / release tag surfaced on GET /health.
		"version": config.Env("APP_VERSION", ""),
		// Application Environment
		//
		// This value determines the "environment" your application is currently
		// running in. This may determine how you prefer to configure various
		// services the application utilizes. Set this in your ".env" file.
		"env": config.Env("APP_ENV", "production"),
		// Application Debug Mode
		"debug": config.Env("APP_DEBUG", false),
		// Disabled framework/custom runners. Supports glob patterns such as goravel:schedule or queue-*.
		"disabled_runners": mergeDisabledRunners(
			parseDisabledRunners(config.GetString("APP_DISABLED_RUNNERS", "")),
			cast.ToString(config.Env("OTEL_TRACES_EXPORTER", "")),
			cast.ToString(config.Env("OTEL_METRICS_EXPORTER", "")),
		),
		// Maintenance mode configuration.
		"maintenance": map[string]any{
			"driver": config.Env("APP_MAINTENANCE_DRIVER", "file"),
			"store":  config.Env("APP_MAINTENANCE_STORE", ""),
		},
		// Allow artisan up/down in production when explicitly enabled (multi-node cache driver recommended).
		"allow_maintenance_commands": config.Env("APP_ALLOW_MAINTENANCE_COMMANDS", false),
		// Enable Development Tool
		"enable_dev_tool": config.Env("APP_ENABLE_DEV_TOOL", false),
		// Application Timezone
		//
		// Here you may specify the default timezone for your application.
		// Example: UTC, Asia/Shanghai
		// More: https://en.wikipedia.org/wiki/List_of_tz_database_time_zones
		"timezone": carbon.UTC,
		// Display source timezone for datetime strings without timezone offset.
		// Used by response time conversion to interpret stored datetime strings.
		"display_source_timezone": config.Env("APP_DISPLAY_SOURCE_TIMEZONE", carbon.UTC),
		// Comma-separated response time field whitelist.
		// Example: created_at,updated_at,deleted_at
		"response_time_fields": config.Env("APP_RESPONSE_TIME_FIELDS", "created_at,updated_at,deleted_at"),
		// Application Locale Configuration
		//
		// The application locale determines the default locale that will be used
		// by the translation service provider. You are free to set this value
		// to any of the locales which will be supported by the application.
		"locale": "en",
		// Application Fallback Locale
		//
		// The fallback locale determines the locale to use when the current one
		// is not available. You may change the value to correspond to any of
		// the language folders that are provided through your application.
		"fallback_locale": "cn",
		// Application Lang Path
		//
		// The path to the language files for the application. You may change
		// the path to a different directory if you would like to customize it.
		// 框架会优先使用 lang_path 指定的文件系统，当文件不存在时才使用 lang_fs embed 文件系统
		"lang_path": "lang",
		"lang_fs":   lang.FS,
		// Encryption Key
		//
		// 32 character string, otherwise these encrypted strings
		// will not be safe. Please do this before deploying an application!
		"key": config.Env("APP_KEY", ""),
	})
}
