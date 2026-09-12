package production

import (
	"strings"

	"github.com/goravel/framework/facades"
)

// WarnInsecureDefaults logs warnings when APP_ENV=production has unsafe settings.
// Does not abort startup — operators must decide; see docs/PRODUCTION.md.
func WarnInsecureDefaults() {
	env := strings.ToLower(strings.TrimSpace(facades.Config().GetString("app.env", "production")))
	if env != "production" {
		return
	}
	logger := facades.Log()
	if logger == nil {
		return
	}

	warn := func(msg string) {
		logger.Warning("[production] " + msg)
	}

	if facades.Config().GetBool("app.debug", false) {
		warn("APP_DEBUG=true — disable in production")
	}
	if strings.TrimSpace(facades.Config().GetString("app.key", "")) == "" {
		warn("APP_KEY is empty — run artisan key:generate")
	}
	if strings.EqualFold(facades.Config().GetString("cache.default", ""), "memory") {
		warn("CACHE_STORE=memory — use redis for multi-instance / rate-limit / locks")
	}
	queue := strings.ToLower(strings.TrimSpace(facades.Config().GetString("queue.default", "sync")))
	if queue == "sync" {
		warn("QUEUE_CONNECTION=sync — async export/jobs will block HTTP workers; use redis/database")
	}
	if facades.Config().GetBool("swagger.enabled", false) {
		warn("SWAGGER_ENABLED=true — disable or protect in production")
	}
	if facades.Config().GetBool("app.enable_dev_tool", false) {
		warn("APP_ENABLE_DEV_TOOL=true — code generator / form demo exposed")
	}
	if facades.Config().GetBool("module.payments_enabled", false) {
		warn("MODULE_PAYMENTS_ENABLED=true — payment gateway is still a stub (501 notify/query)")
	}
	if strings.EqualFold(facades.Config().GetString("tenancy.driver", "off"), "database") {
		if facades.Config().GetBool("tenancy.allow_platform_db_credentials", false) {
			warn("TENANCY_ALLOW_PLATFORM_DB_CREDENTIALS=true — prefer dedicated DB users on public deployments")
		}
		if strings.EqualFold(facades.Config().GetString("tenancy.resolver", "header"), "header") {
			warn("TENANCY_RESOLVER=header — public deployments should prefer subdomain")
		}
	}
}
