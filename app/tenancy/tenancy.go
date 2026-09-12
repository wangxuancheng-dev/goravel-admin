package tenancy

import (
	"context"
	"fmt"
	"strings"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"

	"goravel/app/tenancyctx"
)

// Enabled reports whether database-per-tenant mode is on (TENANCY_DRIVER=database).
func Enabled() bool {
	return strings.EqualFold(facades.Config().GetString("tenancy.driver", "off"), "database")
}

// Resolver returns header | subdomain.
func Resolver() string {
	r := strings.ToLower(strings.TrimSpace(facades.Config().GetString("tenancy.resolver", "header")))
	if r == "subdomain" {
		return "subdomain"
	}
	return "header"
}

// Bound reports whether ctx already carries a tenant ORM connection.
func Bound(ctx context.Context) bool {
	_, ok := tenancyctx.ConnectionFrom(ctx)
	return ok
}

// CacheKey prefixes cache keys with tenant id when tenancy is on and bound.
// When tenancy is on but ctx is unbound, uses a dead-end namespace (never the shared root).
func CacheKey(ctx context.Context, key string) string {
	if !Enabled() {
		return key
	}
	if id, ok := tenancyctx.IDFrom(ctx); ok {
		return fmt.Sprintf("t%d:%s", id, key)
	}
	return "t_unbound:" + key
}

// StoragePrefix returns object-storage path prefix, e.g. tenants/acme/
// When tenancy is on but ctx is unbound, returns tenants/_unbound_/ (never the shared root).
func StoragePrefix(ctx context.Context) string {
	if !Enabled() {
		return ""
	}
	if code, ok := tenancyctx.CodeFrom(ctx); ok {
		return fmt.Sprintf("tenants/%s/", code)
	}
	if id, ok := tenancyctx.IDFrom(ctx); ok {
		return fmt.Sprintf("tenants/%d/", id)
	}
	return "tenants/_unbound_/"
}

// HTTPHint reads tenant id/code from subdomain and/or header/query.
func HTTPHint(ctx http.Context) string {
	if ctx == nil {
		return ""
	}
	if Resolver() == "subdomain" {
		if code := SubdomainHint(ctx.Request().Host()); code != "" {
			return code
		}
	}
	headerName := facades.Config().GetString("tenancy.header", "X-Tenant-ID")
	raw := strings.TrimSpace(ctx.Request().Header(headerName, ""))
	if raw == "" {
		raw = strings.TrimSpace(ctx.Request().Query("tenant_id", ""))
	}
	if raw == "" {
		raw = strings.TrimSpace(ctx.Request().Query("tenant_code", ""))
	}
	return raw
}

// SubdomainHint extracts tenant code from host like acme.example.com.
func SubdomainHint(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}
	if i := strings.Index(host, ":"); i >= 0 {
		host = host[:i]
	}
	host = strings.ToLower(host)
	parts := strings.Split(host, ".")
	if len(parts) < 3 {
		// localhost / bare domain — no tenant subdomain
		return ""
	}
	label := parts[0]
	reserved := strings.Split(facades.Config().GetString("tenancy.subdomain_reserved", "www,api,admin,platform,static,assets"), ",")
	for _, r := range reserved {
		if label == strings.TrimSpace(strings.ToLower(r)) {
			return ""
		}
	}
	return label
}
