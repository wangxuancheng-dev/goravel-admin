package configoverlay

import (
	"testing"
	"time"
)

type mapConfig map[string]any

func (m mapConfig) Env(string, ...any) any             { return nil }
func (m mapConfig) EnvString(string, ...string) string { return "" }
func (m mapConfig) EnvBool(string, ...bool) bool       { return false }
func (m mapConfig) Add(k string, v any)                { m[k] = v }
func (m mapConfig) Get(k string, d ...any) any {
	if v, ok := m[k]; ok {
		return v
	}
	return firstOr(d)
}
func (m mapConfig) GetString(k string, d ...string) string {
	if v, ok := m[k].(string); ok {
		return v
	}
	return firstOr(d)
}
func (m mapConfig) GetInt(k string, d ...int) int {
	if v, ok := m[k].(int); ok {
		return v
	}
	return firstOr(d)
}
func (m mapConfig) GetBool(k string, d ...bool) bool                       { return firstOr(d) }
func (m mapConfig) GetDuration(k string, d ...time.Duration) time.Duration { return firstOr(d) }
func (m mapConfig) GetStringSlice(k string, d ...[]string) []string        { return firstOr(d) }
func (m mapConfig) UnmarshalKey(string, any) error                         { return nil }

func TestOverlayConfigServesOwnConnectionOnly(t *testing.T) {
	base := mapConfig{
		"app.timezone":                       "UTC",
		"database.connections.tenant_1.host": "stale-global-host",
		"database.connections.mysql.host":    "platform-host",
	}
	c := New(base, "database.connections.tenant_1", map[string]any{"host": "tenant-host", "port": 3307})

	if got := c.GetString("database.connections.tenant_1.host"); got != "tenant-host" {
		t.Fatalf("own key = %q", got)
	}
	if got := c.GetInt("database.connections.tenant_1.port"); got != 3307 {
		t.Fatalf("own int key = %d", got)
	}
	// A key of this connection the overlay lacks must not fall through to the global config.
	if got := c.GetString("database.connections.tenant_1.dsn", "dflt"); got != "dflt" {
		t.Fatalf("missing own key = %q, want default", got)
	}
	if got := c.Get("database.connections.tenant_1.read"); got != nil {
		t.Fatalf("missing own key = %v, want nil", got)
	}
	if got := c.GetString("database.connections.mysql.host"); got != "platform-host" {
		t.Fatalf("other connection = %q", got)
	}
	if got := c.GetString("app.timezone"); got != "UTC" {
		t.Fatalf("global key = %q", got)
	}

	c.Add("database.connections.tenant_1.schema", "s1")
	if _, leaked := base["database.connections.tenant_1.schema"]; leaked {
		t.Fatal("Add of an own key must not write the global config")
	}
	if got := c.GetString("database.connections.tenant_1.schema"); got != "s1" {
		t.Fatalf("own key after Add = %q", got)
	}
	whole, ok := c.Get("database.connections.tenant_1").(map[string]any)
	if !ok || whole["host"] != "tenant-host" {
		t.Fatalf("whole connection map = %v", c.Get("database.connections.tenant_1"))
	}
}
