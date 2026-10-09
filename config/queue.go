package config

import (
	"github.com/goravel/framework/contracts/queue"
	"github.com/goravel/framework/facades"
	redisfacades "github.com/goravel/redis/facades"
)

func init() {
	config := facades.Config()
	config.Add("queue", map[string]any{
		// Default Queue Connection Name
		"default": config.Env("QUEUE_CONNECTION", "sync"),
		// Queue Connections
		//
		// Drivers: "sync", "database", "custom" (Redis via goravel/redis).
		// Production: QUEUE_CONNECTION=redis.
		"connections": map[string]any{
			"sync": map[string]any{
				"driver": "sync",
			},
			"database": map[string]any{
				"driver": "database",
				// Follows DB_CONNECTION; override with QUEUE_DATABASE_CONNECTION when needed.
				"connection": config.Env("QUEUE_DATABASE_CONNECTION", config.Env("DB_CONNECTION", "mysql")),
				"queue":      "default",
				"concurrent": 1,
			},
			"redis": map[string]any{
				"driver":     "custom",
				"connection": "default",
				"queue":      "default",
				"via": func() (queue.Driver, error) {
					return redisfacades.Queue("redis")
				},
			},
		},
		// Failed Queue Jobs
		"failed": map[string]any{
			"database": config.Env("DB_CONNECTION", "postgres"),
			"table":    "failed_jobs",
		},
		// Max retries for queue workers (upper bound; per-job ShouldRetry may be lower).
		"tries": config.Env("QUEUE_TRIES", 10),
		// Concurrent = jobs running at once on that logical queue (not process count).
		// QUEUE_CONCURRENT (default queue): 1-2 local/light; 4-8 IO-heavy; keep low for CPU/memory-heavy jobs.
		// QUEUE_LONG_RUNNING_CONCURRENT: usually 1 for export/tenant ops.
		"concurrent":              config.Env("QUEUE_CONCURRENT", 3),
		"long_running_concurrent": config.Env("QUEUE_LONG_RUNNING_CONCURRENT", 1),
		// Search sync logical queue (see config search.sync_queue).
		"search_concurrent": config.Env("QUEUE_SEARCH_CONCURRENT", 2),
		// Per-tenant flexible schedule handlers.
		"schedule_concurrent": config.Env("QUEUE_SCHEDULE_CONCURRENT", 10),
	})
}
