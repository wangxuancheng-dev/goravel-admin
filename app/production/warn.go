package production

import (
	"strings"

	"github.com/goravel/framework/facades"

	"goravel/app/utils"
)

// WarnInsecureDefaults logs warnings when APP_ENV=production has unsafe settings.
// Does not abort startup — operators must decide; see website/docs/deploy/production.md.
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
	if strings.TrimSpace(facades.Config().GetString("jwt.secret", "")) == "" {
		warn("JWT_SECRET is empty — set a strong random secret before public traffic")
	}
	// Cache / queue drivers are enforced (not warned) by RequireRedis before serving.
	if corsAllowsAnyOrigin() {
		warn("CORS_ALLOWED_ORIGINS is empty or * — any website can call the API from a browser; set an explicit allowlist")
	}
	if facades.Config().GetBool("swagger.enabled", false) {
		warn("SWAGGER_ENABLED=true — disable or protect in production")
	}
	if facades.Config().GetBool("app.enable_dev_tool", false) {
		warn("APP_ENABLE_DEV_TOOL=true — code generator / form demo exposed")
	}
	gateways := strings.ToLower(strings.TrimSpace(facades.Config().GetString("module.payment_gateways_enabled", "")))
	if gatewayListed(gateways, "mock") {
		warn("PAYMENT_GATEWAYS_ENABLED lists mock — mock create returns a valid notify sign, so anyone who can create a payment can mark it paid; remove mock from production")
	}
	if gateways == "" || gateways == "*" || gateways == "all" {
		warn("PAYMENT_GATEWAYS_ENABLED is empty/* — every registered gateway except mock is enabled; wechat/alipay notify/query are still stubs until you wire gopay verify; set an explicit list")
	}
	if facades.Config().GetBool("pprof.enabled", false) || facades.Config().GetBool("app.debug", false) {
		token := strings.TrimSpace(facades.Config().GetString("pprof.token", ""))
		ips := strings.TrimSpace(facades.Config().GetString("pprof.allowed_ips", ""))
		if token == "" && ips == "" {
			warn("pprof may be requested via APP_DEBUG/PPROF_ENABLED but PPROF_TOKEN and PPROF_ALLOWED_IPS are empty — routes stay unregistered")
		}
	}
	if strings.EqualFold(facades.Config().GetString("tenancy.driver", "off"), "database") {
		if facades.Config().GetBool("tenancy.allow_platform_db_credentials", false) {
			warn("TENANCY_ALLOW_PLATFORM_DB_CREDENTIALS=true — prefer dedicated DB users on public deployments")
		}
		if !facades.Config().GetBool("tenancy.db_dedicated_user", false) {
			warn("TENANCY_DB_DEDICATED_USER=false — enable for new tenants; backfill existing with artisan tenant:ensure-db-user --all (see production.md minimal tenancy path)")
		}
		if strings.EqualFold(facades.Config().GetString("tenancy.resolver", "header"), "header") {
			warn("TENANCY_RESOLVER=header — public deployments should prefer subdomain")
		}
		if !facades.Config().GetBool("http.forwarded_host_trusted_only", false) {
			warn("TRUSTED_FORWARDED_HOST_ONLY=false — X-Forwarded-Host is taken from any client, so a request can name another tenant's host (its token is still checked against that tenant); set true once your proxies / CDN egress are in TRUSTED_PROXIES")
		}
		if strings.TrimSpace(facades.Config().GetString("tenancy.base_domain", "")) == "" {
			warn("TENANCY_BASE_DOMAIN is empty — no public host is locked against a client-supplied tenant (apex / www / platform accept X-Tenant-ID), and with TENANCY_RESOLVER=subdomain the first label of any host (crm.customer.com) is taken as a tenant code; set it to your public apex")
		}
		mysqlTLS := strings.ToLower(strings.TrimSpace(facades.Config().GetString("tenancy.mysql_tls", "")))
		if mysqlTLS == "" || mysqlTLS == "false" || mysqlTLS == "disable" || mysqlTLS == "off" {
			warn("TENANCY_MYSQL_TLS is empty/off — tenant MySQL connections use plain TCP; set true (or skip-verify) when tenants reach MySQL over the network")
		}
		pgSSL := strings.ToLower(strings.TrimSpace(facades.Config().GetString("tenancy.postgres_sslmode", "")))
		if pgSSL == "" {
			pgSSL = strings.ToLower(strings.TrimSpace(facades.Config().GetString("database.connections.postgres.sslmode", "disable")))
		}
		if pgSSL == "" || pgSSL == "disable" || pgSSL == "allow" {
			warn("PostgreSQL sslmode is disable/allow (TENANCY_POSTGRES_SSLMODE / DB_SSLMODE) — prefer require (or verify-full) for remote tenant databases")
		}
		if disk := strings.TrimSpace(facades.Config().GetString("tenancy.backup_disk", "")); disk == "" {
			warn("TENANT_BACKUP_DISK is empty — tenant backups stay in storage/backups on the host that ran them; with several instances set an object-storage disk (or share that directory)")
		} else if err := utils.ValidateFilesystemDisk(disk); err != nil {
			warn("TENANT_BACKUP_DISK=" + disk + " is not configured (" + err.Error() + ") — tenant backups will fail")
		}
	}
}

// gatewayListed reports whether typ is an exact entry of a comma-separated allowlist.
func gatewayListed(list, typ string) bool {
	for _, part := range strings.Split(list, ",") {
		if strings.TrimSpace(part) == typ {
			return true
		}
	}
	return false
}

// corsAllowsAnyOrigin mirrors the Cors middleware (configStringSlice + resolveCorsOrigin):
// config/cors.go turns an empty CORS_ALLOWED_ORIGINS into ["*"], and only a leading "*"
// allows every Origin. An empty list still falls through to base-domain / tenant checks.
func corsAllowsAnyOrigin() bool {
	var origins []string
	switch v := facades.Config().Get("cors.allowed_origins", []string{"*"}).(type) {
	case []string:
		origins = v
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				origins = append(origins, s)
			}
		}
	default:
		origins = []string{"*"}
	}
	return len(origins) > 0 && origins[0] == "*"
}
