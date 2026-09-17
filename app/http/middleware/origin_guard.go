package middleware

import (
	"strings"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"

	"goravel/app/http/response"
	"goravel/app/services"
	"goravel/app/tenancy"
)

// OriginAllowAdmin rejects browser calls whose Origin host is outside the admin allow set:
// DOMAINS_ADMIN patterns, TENANCY_BASE_DOMAIN (apex + subdomains), and active tenant_domains.
// Empty Origin is allowed (curl / native clients).
// No-op unless multi-tenancy is on and ADMIN_ORIGIN_GUARD is enabled.
func OriginAllowAdmin() http.Middleware {
	return newMiddleware("origin_allow_admin", func(ctx http.Context) {
		if !tenancy.Enabled() || !facades.Config().GetBool("domains.origin_guard_admin", false) {
			ctx.Request().Next()
			return
		}

		origin := strings.TrimSpace(ctx.Request().Header("Origin", ""))
		if origin == "" {
			ctx.Request().Next()
			return
		}

		host := originHost(origin)
		if host == "" {
			response.Abort(ctx, http.StatusForbidden, "origin_not_allowed")
			return
		}

		domains := configStringSlice("domains.admin", nil)
		if isAdminOriginHostAllowed(host, domains) {
			ctx.Request().Next()
			return
		}

		response.Abort(ctx, http.StatusForbidden, "origin_not_allowed")
	})
}

// isAdminOriginHostAllowed reports whether host may call /api/admin from a browser Origin.
// Vanity hosts are checked via tenant_domains (active).
func isAdminOriginHostAllowed(host string, configDomains []string) bool {
	host = tenancy.NormalizeHost(host)
	if host == "" {
		return false
	}

	for _, allowedDomain := range configDomains {
		normalizedAllowed := normalizeHost(allowedDomain)
		if host == normalizedAllowed || matchDomain(host, normalizedAllowed) {
			return true
		}
	}

	if isCorsBaseDomainHost(host) {
		return true
	}

	if tenancy.Enabled() && services.NewTenantDomainService().IsActiveHost(host) {
		return true
	}

	return false
}
