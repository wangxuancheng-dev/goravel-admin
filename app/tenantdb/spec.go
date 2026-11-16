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
	"net/url"
	"strings"
	"time"

	"github.com/goravel/framework/facades"
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
	// MySQL only: go-sql-driver tls= value (true | skip-verify | preferred | custom).
	// Empty disables TLS (plain TCP), matching local Docker defaults.
	TLSMode string

	Pool PoolLimits
}

// Fingerprint identifies the target and credentials: a registered connection whose
// fingerprint differs from a new Spec is rebuilt. Pool limits are not part of it.
func (s Spec) Fingerprint() string {
	h := sha256.New()
	for _, part := range []string{
		strings.ToLower(s.Driver), s.Host, fmt.Sprint(s.Port), s.Database,
		s.Username, s.Password, s.Schema, s.SSLMode, s.TLSMode,
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
		// Goravel's mysql DSN builder has no tls= field; set a full DSN when TLS is on.
		if mode := strings.TrimSpace(s.TLSMode); mode != "" {
			values["dsn"] = s.mysqlDSN(mode)
		}
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

// mysqlDSN matches github.com/goravel/mysql's default DSN shape, plus tls=.
// loc follows database.connections.mysql.loc, then app.timezone (same as the driver).
func (s Spec) mysqlDSN(tlsMode string) string {
	loc := "UTC"
	if facades.Config() != nil {
		loc = strings.TrimSpace(facades.Config().GetString("database.connections.mysql.loc", ""))
		if loc == "" {
			loc = strings.TrimSpace(facades.Config().GetString("app.timezone", "UTC"))
		}
		if loc == "" {
			loc = "UTC"
		}
	}
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true&loc=%s&multiStatements=true&tls=%s",
		s.Username, s.Password, s.Host, s.Port, s.Database, url.QueryEscape(loc), url.QueryEscape(tlsMode))
}
