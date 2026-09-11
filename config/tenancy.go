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
	})
}
