package production

import (
	"fmt"
	"strings"

	"github.com/goravel/framework/facades"
)

// RequireRedis refuses to serve with APP_ENV=production unless cache and queue both
// use redis. With memory cache, rate limits, login lockouts, OnOneServer schedule
// locks, export/import locks and tenant cache invalidation only work inside one
// process; with sync/database queues, jobs either block HTTP or poll the database.
//
// Only the server path calls this: artisan commands (migrate, key:generate, ...) run
// and exit inside bootstrap.Boot, before main reaches it.
func RequireRedis() error {
	env := strings.ToLower(strings.TrimSpace(facades.Config().GetString("app.env", "production")))
	if env != "production" {
		return nil
	}
	var bad []string
	if cache := configValue("cache.default", "memory"); cache != "redis" {
		bad = append(bad, fmt.Sprintf("CACHE_STORE=%s", cache))
	}
	if queue := configValue("queue.default", "sync"); queue != "redis" {
		bad = append(bad, fmt.Sprintf("QUEUE_CONNECTION=%s", queue))
	}
	if len(bad) == 0 {
		return nil
	}
	return fmt.Errorf("APP_ENV=production requires redis: set CACHE_STORE=redis and QUEUE_CONNECTION=redis (got %s)", strings.Join(bad, ", "))
}

func configValue(key, def string) string {
	return strings.ToLower(strings.TrimSpace(facades.Config().GetString(key, def)))
}
