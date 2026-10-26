package config

import (
	"github.com/beratkara/goralens"

	"github.com/goravel/framework/facades"
)

func init() {
	config := facades.Config()
	config.Add("goralens", map[string]any{
		// Debug dashboard for local development only. The package itself defaults
		// to enabled, so keep the fallback false and opt in via .env.
		"enabled": config.Env("GORALENS_ENABLED", false),

		"path": config.Env("GORALENS_PATH", "goralens"),

		"middleware": []goralens.DashboardMiddleware{goralens.Authorize()},

		"allowed_emails": config.Env("GORALENS_ALLOWED_EMAILS", ""),

		"allowed_users": map[string]any{},

		"token_storage_key": config.Env("GORALENS_TOKEN_STORAGE_KEY", ""),

		"session_minutes": config.Env("GORALENS_SESSION_MINUTES", 120),

		// Pin the landlord connection: tenant maintenance swaps the root Orm query
		// to tenant DBs, and flushes on the default connection would land there.
		"storage": map[string]any{
			"driver":     config.Env("GORALENS_DRIVER", "database"),
			"connection": config.Env("GORALENS_DB_CONNECTION", config.Env("DB_CONNECTION", "sqlite")),
		},

		"ignore_paths":    []string{"/favicon.ico"},
		"ignore_commands": []string{},

		"hidden": map[string]any{
			"request_headers":     []string{"authorization", "cookie", "proxy-authorization", "x-csrf-token", "x-xsrf-token", "x-api-key"},
			"response_headers":    []string{"set-cookie"},
			"request_parameters":  []string{"password", "password_confirmation", "current_password", "new_password", "token", "access_token", "refresh_token", "secret", "_token"},
			"response_parameters": []string{"token", "access_token", "refresh_token"},
		},

		// RedisWatcher is omitted: it also needs the separate redishook package.
		"watchers": []goralens.Watcher{
			goralens.RequestWatcher{SizeLimit: 64, IgnoreStatusCodes: []int{}},
			goralens.ClientRequestWatcher{},
			goralens.CommandWatcher{},
			goralens.ScheduleWatcher{},
			goralens.JobWatcher{},
			goralens.EventWatcher{},
			goralens.GateWatcher{IgnoreAbilities: []string{}},
			goralens.MailWatcher{},
			goralens.ExceptionWatcher{},
			goralens.LogWatcher{Level: "debug"},
			goralens.DumpWatcher{},
			goralens.QueryWatcher{SlowMs: 100, Connections: []string{}},
			goralens.ModelWatcher{},
			goralens.CacheWatcher{HiddenKeys: []string{}},
		},
	})
}
