package config

import (
	"github.com/goravel/framework/facades"
)

func init() {
	config := facades.Config()
	config.Add("tenancy", map[string]any{
		// off | database — database 表示一租户一库/Schema（见 docs/TENANT_RESERVED.md）
		"driver": config.Env("TENANCY_DRIVER", "off"),
		// 租户解析：header（X-Tenant-ID / Query）| subdomain（从 Host 取一级子域）
		"resolver": config.Env("TENANCY_RESOLVER", "header"),
		// HTTP Header 名，也可用 Query tenant_id / tenant_code
		"header": config.Env("TENANCY_HEADER", "X-Tenant-ID"),
		// 子域名解析时忽略的标签（逗号分隔）
		"subdomain_reserved": config.Env("TENANCY_SUBDOMAIN_RESERVED", "www,api,admin,platform,static,assets"),
		// 新建租户默认库名前缀（后接 code）
		"database_prefix": config.Env("TENANCY_DATABASE_PREFIX", "tenant_"),
		// PostgreSQL schema 隔离时的 schema 前缀
		"schema_prefix": config.Env("TENANCY_SCHEMA_PREFIX", "tenant_"),
		// 钉死平台连接名（勿指向 tenant_*）；空则首次 PlatformOrmQuery 时取 database.default
		"platform_connection": config.Env("TENANCY_PLATFORM_CONNECTION", ""),
		// 同机开户时是否允许空 username/password 回落平台 DB 账号（远程 host 始终要求独立凭据）
		"allow_platform_db_credentials": config.Env("TENANCY_ALLOW_PLATFORM_DB_CREDENTIALS", true),
		// platform:install 默认管理员（也可传 CLI 参数）
		"platform_admin_username": config.Env("PLATFORM_ADMIN_USERNAME", ""),
		"platform_admin_password": config.Env("PLATFORM_ADMIN_PASSWORD", ""),
		"platform_admin_name":     config.Env("PLATFORM_ADMIN_NAME", ""),
	})
}
