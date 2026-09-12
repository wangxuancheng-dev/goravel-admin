package health

import (
	"context"
	"testing"

	"github.com/goravel/framework/facades"
)

func TestLiveStatus(t *testing.T) {
	prevName := facades.Config().GetString("app.name")
	prevEnv := facades.Config().GetString("app.env")
	prevVer := facades.Config().GetString("app.version")
	facades.Config().Add("app.name", "GoravelAdmin")
	facades.Config().Add("app.env", "test")
	facades.Config().Add("app.version", "1.2.3")
	t.Cleanup(func() {
		facades.Config().Add("app.name", prevName)
		facades.Config().Add("app.env", prevEnv)
		facades.Config().Add("app.version", prevVer)
	})

	r := Live()
	if r.Status != "healthy" {
		t.Fatalf("status=%q", r.Status)
	}
	if r.Timestamp == 0 {
		t.Fatal("timestamp missing")
	}
	if r.App != "GoravelAdmin" || r.Env != "test" || r.Version != "1.2.3" {
		t.Fatalf("meta app=%q env=%q version=%q", r.App, r.Env, r.Version)
	}
}

func TestNeedsRedis(t *testing.T) {
	prevCache := facades.Config().GetString("cache.default")
	prevQueue := facades.Config().GetString("queue.default")
	t.Cleanup(func() {
		facades.Config().Add("cache.default", prevCache)
		facades.Config().Add("queue.default", prevQueue)
	})

	facades.Config().Add("cache.default", "memory")
	facades.Config().Add("queue.default", "sync")
	if needsRedis() {
		t.Fatal("memory+sync should not need redis")
	}

	facades.Config().Add("cache.default", "redis")
	if !needsRedis() {
		t.Fatal("redis cache should need redis")
	}
}

func TestReadyIncludesDatabase(t *testing.T) {
	r := Ready(context.Background())
	if len(r.Checks) == 0 {
		t.Fatal("expected checks")
	}
	found := false
	for _, c := range r.Checks {
		if c.Name == "database" {
			found = true
			// Unit package may run without full ORM boot; only assert shape.
			_ = c.OK
		}
	}
	if !found {
		t.Fatal("database check missing")
	}
}
