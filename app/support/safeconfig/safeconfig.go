// Package safeconfig makes the framework config facade safe for concurrent use.
//
// The framework config is a thin wrapper over viper, which is not goroutine-safe:
// Add (viper.Set) mutates nested maps in place, so an Add racing any Get anywhere
// in the process can abort it with "fatal error: concurrent map read and map write"
// (not recoverable). This app adds config at runtime (tenant connections,
// database.default flips, tenant disks) while requests, queue workers and parallel
// tenant migrations read it, so the container's config instance is replaced with a
// wrapper guarded by an RWMutex.
//
// Install runs from init, and goravel/config blank-imports this package, so the
// wrapper is in place before any app config init() or service provider resolves
// config.
package safeconfig

import (
	"sync"
	"time"

	"github.com/goravel/framework/contracts/binding"
	contractsconfig "github.com/goravel/framework/contracts/config"
	"github.com/goravel/framework/foundation"
)

// Config serializes writes against reads of the wrapped config.
type Config struct {
	mu    sync.RWMutex
	inner contractsconfig.Config
}

var _ contractsconfig.Config = (*Config)(nil)

// Wrap returns c guarded by a lock (c itself if it is already wrapped).
func Wrap(c contractsconfig.Config) contractsconfig.Config {
	if c == nil {
		return nil
	}
	if sc, ok := c.(*Config); ok {
		return sc
	}
	return &Config{inner: c}
}

func init() {
	Install()
}

// Install swaps the container's config instance for the locked wrapper.
// It is idempotent.
func Install() {
	app := foundation.App
	if app == nil {
		return
	}
	current := app.MakeConfig()
	if current == nil {
		return
	}
	if _, ok := current.(*Config); ok {
		return
	}
	app.Instance(binding.Config, Wrap(current))
	// Drop the cached raw instance so the next Make resolves the wrapper.
	app.Fresh(binding.Config)
}

func (c *Config) Env(envName string, defaultValue ...any) any {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.inner.Env(envName, defaultValue...)
}

func (c *Config) EnvString(envName string, defaultValue ...string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.inner.EnvString(envName, defaultValue...)
}

func (c *Config) EnvBool(envName string, defaultValue ...bool) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.inner.EnvBool(envName, defaultValue...)
}

func (c *Config) Add(name string, configuration any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.inner.Add(name, configuration)
}

func (c *Config) Get(path string, defaultValue ...any) any {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.inner.Get(path, defaultValue...)
}

func (c *Config) GetString(path string, defaultValue ...string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.inner.GetString(path, defaultValue...)
}

func (c *Config) GetInt(path string, defaultValue ...int) int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.inner.GetInt(path, defaultValue...)
}

func (c *Config) GetBool(path string, defaultValue ...bool) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.inner.GetBool(path, defaultValue...)
}

func (c *Config) GetDuration(path string, defaultValue ...time.Duration) time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.inner.GetDuration(path, defaultValue...)
}

func (c *Config) GetStringSlice(path string, defaultValue ...[]string) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.inner.GetStringSlice(path, defaultValue...)
}

func (c *Config) UnmarshalKey(key string, rawVal any) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.inner.UnmarshalKey(key, rawVal)
}
