package config

import (
	"crypto/tls"
	"strings"

	"github.com/goravel/framework/contracts/config"
	"github.com/goravel/framework/contracts/database/driver"
	"github.com/goravel/framework/facades"
	mysqlfacades "github.com/goravel/mysql/facades"
	postgresfacades "github.com/goravel/postgres/facades"
	"github.com/spf13/cast"
)

func init() {
	config := facades.Config()
	config.Add("database", map[string]any{
		// Default database connection name
		"default": config.Env("DB_CONNECTION", "sqlite"),
		// Database connections
		"connections": map[string]any{
			// "sqlserver": map[string]any{
			// 	"host":     config.Env("DB_HOST", "127.0.0.1"),
			// 	"port":     config.Env("DB_PORT", 3306),
			// 	"database": config.Env("DB_DATABASE", "forge"),
			// 	"username": config.Env("DB_USERNAME", ""),
			// 	"password": config.Env("DB_PASSWORD", ""),
			// 	"charset":  "utf8mb4",
			// 	"prefix":   "",
			// 	"singular": false,
			// 	"via": func() (driver.Driver, error) {
			// 		return sqlserverfacades.Sqlserver("sqlserver")
			// 	},
			// },
			// "sqlite": map[string]any{
			// 	"database": config.Env("DB_DATABASE", "forge"),
			// 	"prefix":   "",
			// 	"singular": false,
			// 	"via": func() (driver.Driver, error) {
			// 		return sqlitefacades.Sqlite("sqlite")
			// 	},
			// },
			"postgres": map[string]any{
				"host":     config.Env("DB_HOST", "127.0.0.1"),
				"port":     config.Env("DB_PORT", 5432),
				"database": config.Env("DB_DATABASE", "forge"),
				"username": config.Env("DB_USERNAME", ""),
				"password": config.Env("DB_PASSWORD", ""),
			"sslmode":  config.Env("DB_SSLMODE", "disable"),
			"singular": false,
			"prefix":   "",
			"schema":   config.Env("DB_SCHEMA", "public"),
			"via": func() (driver.Driver, error) {
				return postgresfacades.Postgres("postgres")
			},
		},
			"mysql": map[string]any{
				"host":      config.Env("DB_HOST", "127.0.0.1"),
				"port":      config.Env("DB_PORT", 3306),
				"database":  config.Env("DB_DATABASE", "forge"),
				"username":  config.Env("DB_USERNAME", ""),
				"password":  config.Env("DB_PASSWORD", ""),
				"charset":   "utf8mb4",
				"collation": "utf8mb4_unicode_ci",
				"prefix":    "",
				"singular":  false,
				"via": func() (driver.Driver, error) {
					return mysqlfacades.Mysql("mysql")
				},
			},
		},
		// Set pool configuration
		"pool": map[string]any{
			// Sets the maximum number of connections in the idle
			// connection pool.
			//
			// If MaxOpenConns is greater than 0 but less than the new MaxIdleConns,
			// then the new MaxIdleConns will be reduced to match the MaxOpenConns limit.
			//
			// If n <= 0, no idle connections are retained.
			"max_idle_conns": 10,
			// Sets the maximum number of open connections to the database.
			//
			// If MaxIdleConns is greater than 0 and the new MaxOpenConns is less than
			// MaxIdleConns, then MaxIdleConns will be reduced to match the new
			// MaxOpenConns limit.
			//
			// If n <= 0, then there is no limit on the number of open connections.
			"max_open_conns": 100,
			// Sets the maximum amount of time a connection may be idle.
			//
			// Expired connections may be closed lazily before reuse.
			//
			// If d <= 0, connections are not closed due to a connection's idle time.
			// Unit: Second
			"conn_max_idletime": 3600,
			// Sets the maximum amount of time a connection may be reused.
			//
			// Expired connections may be closed lazily before reuse.
			//
			// If d <= 0, connections are not closed due to a connection's age.
			// Unit: Second
			"conn_max_lifetime": 3600,
		},
		// Migration Repository Table
		//
		// This table keeps track of all the migrations that have already run for
		// your application. Using this information, we can determine which of
		// the migrations on disk haven't actually been run in the database.
		// Available Drivers: "default", "sql"
		"migrations": map[string]any{
			"driver": "default",
			"table":  "migrations",
		},
		"redis": map[string]any{
			"default": redisDefaultConnection(config),
		},
	})
}

// redisDefaultConnection builds database.redis.default.
// Plain Redis (local/Docker): leave REDIS_TLS unset/false — no tls field.
// TLS hosts (e.g. AWS ElastiCache rediss): REDIS_TLS=true; optional REDIS_USERNAME / REDIS_CLUSTER.
func redisDefaultConnection(c config.Config) map[string]any {
	host := cast.ToString(c.Env("REDIS_HOST", ""))
	conn := map[string]any{
		"host":     host,
		"username": cast.ToString(c.Env("REDIS_USERNAME", "")),
		"password": cast.ToString(c.Env("REDIS_PASSWORD", "")),
		"port":     c.Env("REDIS_PORT", 6379),
		"database": c.Env("REDIS_DB", 0),
		"cluster":  cast.ToBool(c.Env("REDIS_CLUSTER", false)),
	}
	if tlsCfg := redisTLSConfig(
		cast.ToBool(c.Env("REDIS_TLS", false)),
		host,
		cast.ToString(c.Env("REDIS_TLS_SERVER_NAME", "")),
		cast.ToBool(c.Env("REDIS_TLS_INSECURE_SKIP_VERIFY", false)),
	); tlsCfg != nil {
		conn["tls"] = tlsCfg
	}
	return conn
}

// redisTLSConfig returns nil when TLS is disabled so go-redis stays on plain TCP.
func redisTLSConfig(enabled bool, host, serverName string, insecureSkipVerify bool) *tls.Config {
	if !enabled {
		return nil
	}
	cfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
	name := strings.TrimSpace(serverName)
	if name == "" {
		name = strings.TrimSpace(host)
	}
	if name != "" {
		cfg.ServerName = name
	}
	if insecureSkipVerify {
		cfg.InsecureSkipVerify = true
	}
	return cfg
}
