package config

import (
	"github.com/goravel/framework/facades"
)

func init() {
	config := facades.Config()
	config.Add("tenancy", map[string]any{
		// off | database — database 表示一租户一库/Schema（见 website/docs/advanced/tenancy.md）
		"driver": config.Env("TENANCY_DRIVER", "off"),
		// 租户解析：header（X-Tenant-ID / Query）| subdomain（公网推荐）
		"resolver": config.Env("TENANCY_RESOLVER", "header"),
		// HTTP Header 名，也可用 Query tenant_id / tenant_code
		"header": config.Env("TENANCY_HEADER", "X-Tenant-ID"),
		// 子域模式且 Host 解析不到租户时，是否允许回落 Header/Query/body（空=subdomain 默认 false，header 默认 true）
		"allow_header_fallback": config.Env("TENANCY_ALLOW_HEADER_FALLBACK", ""),
		// 子域名解析时忽略的标签（逗号分隔）
		"subdomain_reserved": config.Env("TENANCY_SUBDOMAIN_RESERVED", "www,api,admin,platform,static,assets"),
		// 新建租户默认库名前缀（后接 code）
		"database_prefix": config.Env("TENANCY_DATABASE_PREFIX", "tenant_"),
		// PostgreSQL schema 隔离时的 schema 前缀
		"schema_prefix": config.Env("TENANCY_SCHEMA_PREFIX", "tenant_"),
		// 钉死平台连接名（勿指向 tenant_*）；空则首次 PlatformOrmQuery 时取 database.default
		"platform_connection": config.Env("TENANCY_PLATFORM_CONNECTION", ""),
		// 同机开户是否允许空账号回落平台 DB_*（公网默认 false；本地开发可 true）
		"allow_platform_db_credentials": config.Env("TENANCY_ALLOW_PLATFORM_DB_CREDENTIALS", false),
		// PostgreSQL 租户连接 sslmode（也可由 database.connections.postgres.sslmode / DB_SSLMODE 提供）
		"postgres_sslmode": config.Env("TENANCY_POSTGRES_SSLMODE", ""),
		// tenant:backup 每个租户保留的最近份数（0=不清理）
		"backup_keep": config.Env("TENANCY_BACKUP_KEEP", 10),
		// 每租户连接池（公网收紧；覆盖 database.pool 对动态连接的默认）
		"pool_max_idle_conns":   config.Env("TENANCY_POOL_MAX_IDLE_CONNS", 2),
		"pool_max_open_conns":   config.Env("TENANCY_POOL_MAX_OPEN_CONNS", 20),
		"pool_conn_max_idletime": config.Env("TENANCY_POOL_CONN_MAX_IDLETIME", 300),
		"pool_conn_max_lifetime": config.Env("TENANCY_POOL_CONN_MAX_LIFETIME", 1800),
		// platform:install 默认管理员（也可传 CLI 参数）
		"platform_admin_username": config.Env("PLATFORM_ADMIN_USERNAME", ""),
		"platform_admin_password": config.Env("PLATFORM_ADMIN_PASSWORD", ""),
		"platform_admin_name":     config.Env("PLATFORM_ADMIN_NAME", ""),
	})
}
