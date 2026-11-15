package bootstrap

import (
	"strconv"
	"sync"
	"testing"

	"github.com/goravel/framework/facades"

	"goravel/app/support/safeconfig"
)

// Booting through goravel/config must leave the locked wrapper as the config facade.
func TestConfigFacadeIsGoroutineSafe(t *testing.T) {
	cfg := facades.Config()
	if _, ok := cfg.(*safeconfig.Config); !ok {
		t.Fatalf("facades.Config() is %T, want *safeconfig.Config", cfg)
	}

	// Same shape as tenant registration: nested Add while other goroutines read.
	var wg sync.WaitGroup
	for w := 0; w < 8; w++ {
		wg.Add(2)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < 500; i++ {
				cfg.Add("safeconfig_stress.c"+strconv.Itoa(w*500+i), map[string]any{"database": "db"})
			}
		}(w)
		go func() {
			defer wg.Done()
			for i := 0; i < 500; i++ {
				_ = cfg.GetString("safeconfig_stress.c" + strconv.Itoa(i) + ".database")
				_ = cfg.GetString("database.default")
			}
		}()
	}
	wg.Wait()

	if got := cfg.GetString("safeconfig_stress.c42.database"); got != "db" {
		t.Fatalf("got %q, want db", got)
	}
}
