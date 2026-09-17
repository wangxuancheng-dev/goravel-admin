package config

import (
	"strings"

	"github.com/goravel/framework/facades"
)

func init() {
	config := facades.Config()

	// 解析域名字符串的辅助函数
	parseDomains := func(envValue string) []string {
		if envValue == "" {
			return []string{} // 空数组表示不限制域名
		}
		// 分割逗号分隔的域名，并去除空格
		domains := strings.Split(envValue, ",")
		allowedDomains := make([]string, 0, len(domains))
		for _, domain := range domains {
			domain = strings.TrimSpace(domain)
			if domain != "" {
				allowedDomains = append(allowedDomains, domain)
			}
		}
		return allowedDomains
	}

	config.Add("domains", map[string]any{
		// Admin Routes Allowed Domains
		//
		// The domains that are allowed to access admin routes (/api/admin/*).
		// If not set or empty, all domains are allowed.
		// You can set multiple domains separated by commas in the .env file.
		// Supports wildcard patterns like *.example.com
		//
		// Examples in .env file:
		//   Single domain: DOMAINS_ADMIN=admin.example.com
		//   Multiple domains: DOMAINS_ADMIN=admin.example.com,*.admin.example.com
		//   Multiple domains with spaces: DOMAINS_ADMIN=admin.example.com, *.admin.example.com
		//   Disable domain restriction: DOMAINS_ADMIN= (empty or not set)
		"admin": func() []string {
			domainsEnv := config.Env("DOMAINS_ADMIN", "")
			domainsStr := ""
			if str, ok := domainsEnv.(string); ok {
				domainsStr = str
			}
			return parseDomains(domainsStr)
		}(),

		// Open API Routes Allowed Domains
		//
		// The domains that are allowed to access open API routes (/api/open/*).
		// If not set or empty, all domains are allowed.
		//
		// Examples in .env file:
		//   DOMAINS_OPEN=open.example.com,*.open.example.com
		"open": func() []string {
			domainsEnv := config.Env("DOMAINS_OPEN", "")
			domainsStr := ""
			if str, ok := domainsEnv.(string); ok {
				domainsStr = str
			}
			return parseDomains(domainsStr)
		}(),

		// API Routes Allowed Domains
		//
		// The domains that are allowed to access general API routes (/api/*).
		// If not set or empty, all domains are allowed.
		//
		// Examples in .env file:
		//   DOMAINS_API=api.example.com,*.api.example.com
		"api": func() []string {
			domainsEnv := config.Env("DOMAINS_API", "")
			domainsStr := ""
			if str, ok := domainsEnv.(string); ok {
				domainsStr = str
			}
			return parseDomains(domainsStr)
		}(),

		// When true (and tenancy is on), /api/admin rejects browser Origin hosts outside
		// DOMAINS_ADMIN, TENANCY_BASE_DOMAIN (subdomains), and active tenant_domains.
		// Empty Origin is allowed. Ignored when TENANCY_DRIVER=off.
		"origin_guard_admin": func() bool {
			raw := config.Env("ADMIN_ORIGIN_GUARD", false)
			switch v := raw.(type) {
			case bool:
				return v
			case string:
				s := strings.ToLower(strings.TrimSpace(v))
				return s == "1" || s == "true" || s == "yes" || s == "on"
			default:
				return false
			}
		}(),
	})
}
