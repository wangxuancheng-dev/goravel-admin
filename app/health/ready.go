package health

import (
	"context"
	"strings"
	"time"

	"github.com/goravel/framework/facades"

	"goravel/app/clients"
	appfacades "goravel/app/facades"
)

// Check is one readiness probe result.
type Check struct {
	Name    string `json:"name"`
	OK      bool   `json:"ok"`
	Detail  string `json:"detail,omitempty"`
	Skipped bool   `json:"skipped,omitempty"`
}

// Report aggregates liveness/readiness.
type Report struct {
	Status    string  `json:"status"` // healthy | ready | not_ready
	Timestamp int64   `json:"timestamp"`
	Checks    []Check `json:"checks,omitempty"`
}

// Live always reports process-up (for load balancer liveness).
func Live() Report {
	return Report{
		Status:    "healthy",
		Timestamp: time.Now().Unix(),
	}
}

// Ready pings DB and, when configured, Redis used by cache/queue.
func Ready(ctx context.Context) Report {
	if ctx == nil {
		ctx = context.Background()
	}
	checks := []Check{checkDatabase(ctx)}
	if needsRedis() {
		checks = append(checks, checkRedis(ctx))
	} else {
		checks = append(checks, Check{
			Name:    "redis",
			OK:      true,
			Skipped: true,
			Detail:  "cache/queue not using redis",
		})
	}

	ok := true
	for _, c := range checks {
		if !c.Skipped && !c.OK {
			ok = false
			break
		}
	}
	status := "ready"
	if !ok {
		status = "not_ready"
	}
	return Report{
		Status:    status,
		Timestamp: time.Now().Unix(),
		Checks:    checks,
	}
}

func needsRedis() bool {
	cacheStore := strings.ToLower(strings.TrimSpace(facades.Config().GetString("cache.default", "memory")))
	if cacheStore == "redis" {
		return true
	}
	queue := strings.ToLower(strings.TrimSpace(facades.Config().GetString("queue.default", "sync")))
	switch queue {
	case "redis", "redisstream", "redis_stream":
		return true
	default:
		return false
	}
}

func checkDatabase(parent context.Context) Check {
	ctx, cancel := context.WithTimeout(parent, 2*time.Second)
	defer cancel()
	orm := appfacades.Orm()
	if orm == nil {
		return Check{Name: "database", OK: false, Detail: "orm unavailable"}
	}
	sqlDB, err := orm.DB()
	if err != nil {
		return Check{Name: "database", OK: false, Detail: err.Error()}
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		return Check{Name: "database", OK: false, Detail: err.Error()}
	}
	name := facades.Config().GetString("database.default", "")
	return Check{Name: "database", OK: true, Detail: name}
}

func checkRedis(parent context.Context) Check {
	ctx, cancel := context.WithTimeout(parent, 2*time.Second)
	defer cancel()
	client, err := clients.GetRedisClient("default")
	if err != nil {
		return Check{Name: "redis", OK: false, Detail: err.Error()}
	}
	if err := client.Ping(ctx).Err(); err != nil {
		return Check{Name: "redis", OK: false, Detail: err.Error()}
	}
	return Check{Name: "redis", OK: true}
}
