package safeconfig

import (
	"strconv"
	"sync"
	"testing"
	"time"

	contractsconfig "github.com/goravel/framework/contracts/config"
	"github.com/goravel/framework/facades"
)

// mapConfig is deliberately not goroutine-safe: without the wrapper's lock the
// stress test below dies with "concurrent map read and map write".
type mapConfig struct{ m map[string]any }

func (c *mapConfig) Env(string, ...any) any             { return nil }
func (c *mapConfig) EnvString(string, ...string) string { return "" }
func (c *mapConfig) EnvBool(string, ...bool) bool       { return false }
func (c *mapConfig) Add(name string, v any)             { c.m[name] = v }
func (c *mapConfig) Get(p string, _ ...any) any         { return c.m[p] }
func (c *mapConfig) GetString(p string, _ ...string) string {
	s, _ := c.m[p].(string)
	return s
}
func (c *mapConfig) GetInt(p string, _ ...int) int {
	i, _ := c.m[p].(int)
	return i
}
func (c *mapConfig) GetBool(p string, _ ...bool) bool {
	b, _ := c.m[p].(bool)
	return b
}
func (c *mapConfig) GetDuration(p string, _ ...time.Duration) time.Duration {
	d, _ := c.m[p].(time.Duration)
	return d
}
func (c *mapConfig) GetStringSlice(p string, _ ...[]string) []string {
	s, _ := c.m[p].([]string)
	return s
}
func (c *mapConfig) UnmarshalKey(string, any) error { return nil }

var _ contractsconfig.Config = (*mapConfig)(nil)

func TestWrapSerializesConcurrentAddAndGet(t *testing.T) {
	cfg := Wrap(&mapConfig{m: map[string]any{}})

	var wg sync.WaitGroup
	for w := 0; w < 8; w++ {
		wg.Add(2)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < 5000; i++ {
				cfg.Add("k"+strconv.Itoa(w*5000+i), i)
			}
		}(w)
		go func() {
			defer wg.Done()
			for i := 0; i < 5000; i++ {
				_ = cfg.GetInt("k" + strconv.Itoa(i))
				_ = cfg.GetString("k" + strconv.Itoa(i))
			}
		}()
	}
	wg.Wait()

	if got := cfg.GetInt("k4999"); got != 4999 {
		t.Fatalf("GetInt(k4999) = %d, want 4999", got)
	}
}

func TestWrapIsIdempotent(t *testing.T) {
	inner := &mapConfig{m: map[string]any{}}
	w := Wrap(inner)
	if Wrap(w) != w {
		t.Fatal("Wrap of a wrapped config must return it unchanged")
	}
	if Wrap(nil) != nil {
		t.Fatal("Wrap(nil) must be nil")
	}
}

func TestInstallReplacesFacade(t *testing.T) {
	Install()
	Install()
	if _, ok := facades.Config().(*Config); !ok {
		t.Fatalf("facades.Config() is %T, want *safeconfig.Config", facades.Config())
	}
}
