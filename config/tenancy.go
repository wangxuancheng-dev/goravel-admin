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
		// tenant:backup keep count (0 = never prune)
		"backup_keep": config.Env("TENANT_BACKUP_KEEP", 10),
		// When true, tenant:backup-scheduled (kernel DailyAt) actually runs backups
		"backup_schedule_enabled": config.Env("TENANT_BACKUP_SCHEDULE_ENABLED", false),
		// UTC HH:MM for tenant:backup-scheduled DailyAt
		"backup_schedule_at": config.Env("TENANT_BACKUP_SCHEDULE_AT", "20:00"),
		// Soft-deleted tenant retention days before tenant:cleanup-deleted hard-deletes (0=never auto)
		"deleted_retention_days": config.Env("TENANCY_DELETED_RETENTION_DAYS", 30),
		// 每租户连接池（公网收紧；覆盖 database.pool 对动态连接的默认）
		"pool_max_idle_conns":    config.Env("TENANCY_POOL_MAX_IDLE_CONNS", 2),
		"pool_max_open_conns":    config.Env("TENANCY_POOL_MAX_OPEN_CONNS", 20),
		"pool_conn_max_idletime": config.Env("TENANCY_POOL_CONN_MAX_IDLETIME", 300),
		"pool_conn_max_lifetime": config.Env("TENANCY_POOL_CONN_MAX_LIFETIME", 1800),
		// Process-local registered tenant pools: 0 max = unlimited; idle TTL seconds (0 = never idle-evict)
		"registered_max":      config.Env("TENANCY_REGISTERED_MAX", 0),
		"registered_idle_ttl": config.Env("TENANCY_REGISTERED_IDLE_TTL", 900),
		// Fleet scope: 0 batch = unlimited unless fleet > auto_batch_at
		"scope_batch":          config.Env("TENANCY_SCOPE_BATCH", 0),
		"scope_auto_batch_at":  config.Env("TENANCY_SCOPE_AUTO_BATCH_AT", 200),
		"scope_auto_batch":     config.Env("TENANCY_SCOPE_AUTO_BATCH", 100),
		// flexible-schedule:tick max due rows enqueued per minute (fan-out only)
		"flex_schedule_tick_limit": config.Env("TENANCY_FLEX_SCHEDULE_TICK_LIMIT", 2000),
		// logical queue name for FlexibleScheduleRun jobs
		"flex_schedule_queue": config.Env("TENANCY_FLEX_SCHEDULE_QUEUE", "schedule"),
		// platform:install 默认管理员（也可传 CLI 参数）
		"platform_admin_username": config.Env("PLATFORM_ADMIN_USERNAME", ""),
		"platform_admin_password": config.Env("PLATFORM_ADMIN_PASSWORD", ""),
		"platform_admin_name":     config.Env("PLATFORM_ADMIN_NAME", ""),
		// Public apex for subdomain URLs (acme.{base_domain}); empty disables subdomain URL helpers
		"base_domain": config.Env("TENANCY_BASE_DOMAIN", ""),
		// Unified ingress hostname customers CNAME to (edge SSL mode)
		"domain_target": config.Env("TENANCY_DOMAIN_TARGET", ""),
		// DNS TXT host prefix: _{token_prefix}.{custom_host}
		"domain_verify_prefix": config.Env("TENANCY_DOMAIN_VERIFY_PREFIX", "_goravel-tenant"),
		// Cache TTL seconds for host -> tenant code (0 = 60)
		"domain_cache_ttl": config.Env("TENANCY_DOMAIN_CACHE_TTL", 60),
	})
}
