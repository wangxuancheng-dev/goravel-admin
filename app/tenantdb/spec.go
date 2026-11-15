// Package tenantdb owns the database connections of tenants (database-per-tenant).
//
// Tenant connections are built from the framework's exported constructors with a
// per-connection config overlay, so they never touch the global config, the framework
// Orm connection cache or the driver-level gorm cache. The app therefore decides when
// a pool is created, replaced and closed, instead of working around caches it cannot
// invalidate. Lookups on the request path are lock-free reads.
package tenantdb

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// Supported drivers.
const (
	DriverMySQL    = "mysql"
	DriverPostgres = "postgres"
)

// PoolLimits configures the sql.DB pool of one tenant connection.
type PoolLimits struct {
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxIdleTime time.Duration
	ConnMaxLifetime time.Duration
}

// Spec describes how to reach one tenant database.
type Spec struct {
	Driver   string // DriverMySQL | DriverPostgres
	Host     string
	Port     int
	Database string
	Username string
	Password string
	// Postgres only.
	Schema  string
	SSLMode string

	Pool PoolLimits
}

// Fingerprint identifies the target and credentials: a registered connection whose
// fingerprint differs from a new Spec is rebuilt. Pool limits are not part of it.
func (s Spec) Fingerprint() string {
	h := sha256.New()
	for _, part := range []string{
		strings.ToLower(s.Driver), s.Host, fmt.Sprint(s.Port), s.Database,
		s.Username, s.Password, s.Schema, s.SSLMode,
	} {
		h.Write([]byte(part))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

// Validate reports a Spec that cannot be built.
func (s Spec) Validate() error {
	switch s.Driver {
	case DriverMySQL, DriverPostgres:
	default:
		return fmt.Errorf("tenantdb: unsupported driver %q", s.Driver)
	}
	if strings.TrimSpace(s.Database) == "" {
		return fmt.Errorf("tenantdb: database name is required")
	}
	return nil
}

// connectionValues are the database.connections.<name>.* keys the goravel mysql /
// postgres drivers read.
func (s Spec) connectionValues() map[string]any {
	values := map[string]any{
		"driver":   s.Driver,
		"host":     s.Host,
		"port":     s.Port,
		"database": s.Database,
		"username": s.Username,
		"password": s.Password,
		"prefix":   "",
		"singular": false,
	}
	switch s.Driver {
	case DriverMySQL:
		values["charset"] = "utf8mb4"
		values["collation"] = "utf8mb4_unicode_ci"
	case DriverPostgres:
		schema := s.Schema
		if schema == "" {
			schema = "public"
		}
		values["schema"] = schema
		values["sslmode"] = s.SSLMode
	}
	return values
}
