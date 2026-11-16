package production

import (
	"strings"
	"testing"

	"github.com/goravel/framework/facades"
)

func TestRequireRedis(t *testing.T) {
	prevEnv := facades.Config().GetString("app.env")
	prevCache := facades.Config().GetString("cache.default")
	prevQueue := facades.Config().GetString("queue.default")
	t.Cleanup(func() {
		facades.Config().Add("app.env", prevEnv)
		facades.Config().Add("cache.default", prevCache)
		facades.Config().Add("queue.default", prevQueue)
	})

	cases := []struct {
		name, env, cache, queue string
		wantErr                 []string
	}{
		{"local allows anything", "local", "memory", "sync", nil},
		{"production redis ok", "production", "redis", "redis", nil},
		{"production case-insensitive", " Production ", "Redis", "REDIS", nil},
		{"production memory cache", "production", "memory", "redis", []string{"CACHE_STORE=memory"}},
		{"production database queue", "production", "redis", "database", []string{"QUEUE_CONNECTION=database"}},
		{"production both wrong", "production", "memory", "sync", []string{"CACHE_STORE=memory", "QUEUE_CONNECTION=sync"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			facades.Config().Add("app.env", tc.env)
			facades.Config().Add("cache.default", tc.cache)
			facades.Config().Add("queue.default", tc.queue)
			err := RequireRedis()
			if len(tc.wantErr) == 0 {
				if err != nil {
					t.Fatalf("RequireRedis() = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatal("RequireRedis() = nil, want error")
			}
			for _, s := range tc.wantErr {
				if !strings.Contains(err.Error(), s) {
					t.Fatalf("error %q does not mention %q", err, s)
				}
			}
		})
	}
}
