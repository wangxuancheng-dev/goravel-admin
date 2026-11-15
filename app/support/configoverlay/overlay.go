// Package configoverlay serves the settings of one runtime-defined resource (a tenant
// database connection, a tenant storage disk) from its own map, without writing them
// into the process-wide config.
//
// Framework drivers read their settings through a config.Config by key
// (e.g. "database.connections.<name>.host", "filesystems.disks.<name>.bucket"). An
// Overlay answers the keys under its root from its own values and delegates every
// other key to the base config, so a driver built with it sees a fully configured
// resource that nothing else in the process can see or race on.
package configoverlay

import (
	"strings"
	"sync"
	"time"

	contractsconfig "github.com/goravel/framework/contracts/config"
	"github.com/spf13/cast"
)

// Overlay is a config.Config that owns the keys under one root.
type Overlay struct {
	base   contractsconfig.Config
	root   string
	prefix string

	mu     sync.RWMutex
	values map[string]any
}

var _ contractsconfig.Config = (*Overlay)(nil)

// New returns an Overlay owning root (e.g. "filesystems.disks.tenant_byob_7"), with a
// copy of values as the keys below it.
func New(base contractsconfig.Config, root string, values map[string]any) *Overlay {
	copied := make(map[string]any, len(values))
	for k, v := range values {
		copied[k] = v
	}
	return &Overlay{base: base, root: root, prefix: root + ".", values: copied}
}

// Set sets a key below the root (e.g. "via").
func (c *Overlay) Set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values[key] = value
}

// lookup reports whether path belongs to the root; when it does, value is the
// overlay's value (nil, false for a key the overlay does not have — never the base's).
func (c *Overlay) lookup(path string) (owned bool, value any, found bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if path == c.root {
		all := make(map[string]any, len(c.values))
		for k, v := range c.values {
			all[k] = v
		}
		return true, all, true
	}
	key, ok := strings.CutPrefix(path, c.prefix)
	if !ok {
		return false, nil, false
	}
	value, found = c.values[key]
	return true, value, found
}

func firstOr[T any](defaults []T) T {
	var zero T
	if len(defaults) > 0 {
		return defaults[0]
	}
	return zero
}

func (c *Overlay) Env(envName string, defaultValue ...any) any {
	return c.base.Env(envName, defaultValue...)
}

func (c *Overlay) EnvString(envName string, defaultValue ...string) string {
	return c.base.EnvString(envName, defaultValue...)
}

func (c *Overlay) EnvBool(envName string, defaultValue ...bool) bool {
	return c.base.EnvBool(envName, defaultValue...)
}

// Add keeps keys below the root local; anything else goes to the base config.
func (c *Overlay) Add(name string, configuration any) {
	if key, ok := strings.CutPrefix(name, c.prefix); ok {
		c.Set(key, configuration)
		return
	}
	c.base.Add(name, configuration)
}

func (c *Overlay) Get(path string, defaultValue ...any) any {
	if owned, v, found := c.lookup(path); owned {
		if found {
			return v
		}
		return firstOr(defaultValue)
	}
	return c.base.Get(path, defaultValue...)
}

func (c *Overlay) GetString(path string, defaultValue ...string) string {
	if owned, v, found := c.lookup(path); owned {
		if found {
			return cast.ToString(v)
		}
		return firstOr(defaultValue)
	}
	return c.base.GetString(path, defaultValue...)
}

func (c *Overlay) GetInt(path string, defaultValue ...int) int {
	if owned, v, found := c.lookup(path); owned {
		if found {
			return cast.ToInt(v)
		}
		return firstOr(defaultValue)
	}
	return c.base.GetInt(path, defaultValue...)
}

func (c *Overlay) GetBool(path string, defaultValue ...bool) bool {
	if owned, v, found := c.lookup(path); owned {
		if found {
			return cast.ToBool(v)
		}
		return firstOr(defaultValue)
	}
	return c.base.GetBool(path, defaultValue...)
}

func (c *Overlay) GetDuration(path string, defaultValue ...time.Duration) time.Duration {
	if owned, v, found := c.lookup(path); owned {
		if found {
			return cast.ToDuration(v)
		}
		return firstOr(defaultValue)
	}
	return c.base.GetDuration(path, defaultValue...)
}

func (c *Overlay) GetStringSlice(path string, defaultValue ...[]string) []string {
	if owned, v, found := c.lookup(path); owned {
		if found {
			return cast.ToStringSlice(v)
		}
		return firstOr(defaultValue)
	}
	return c.base.GetStringSlice(path, defaultValue...)
}

func (c *Overlay) UnmarshalKey(key string, rawVal any) error {
	return c.base.UnmarshalKey(key, rawVal)
}
