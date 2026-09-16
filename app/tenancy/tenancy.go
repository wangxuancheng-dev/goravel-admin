package tenancy

import (
	"context"
	"fmt"
	"strings"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"

	apperrors "goravel/app/errors"
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

// AllowHeaderFallback reports whether Header/Query/body hints may be used when
// subdomain resolution yields nothing. Default: false for subdomain (public-safe),
// true for header resolver (local/SPA DX). Override with TENANCY_ALLOW_HEADER_FALLBACK.
func AllowHeaderFallback() bool {
	raw := strings.TrimSpace(facades.Config().GetString("tenancy.allow_header_fallback", ""))
	if raw != "" {
		switch strings.ToLower(raw) {
		case "1", "true", "yes", "on":
			return true
		default:
			return false
		}
	}
	return Resolver() != "subdomain"
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

// ClientHint reads tenant id/code from Header / Query only (never Host).
func ClientHint(ctx http.Context) string {
	if ctx == nil {
		return ""
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

// HTTPHint resolves the tenant hint for middleware (no explicit body override).
func HTTPHint(ctx http.Context) string {
	hint, _ := ResolveHint(ctx, "")
	return hint
}

// ResolveHint picks the effective tenant code/id for binding.
// Priority for subdomain resolver: Host subdomain > (optional) client/body hint.
// When subdomain is present and clientHint conflicts, returns ErrTenantHintConflict.
func ResolveHint(ctx http.Context, clientHint string) (string, error) {
	clientHint = strings.TrimSpace(clientHint)
	if clientHint == "" && ctx != nil {
		clientHint = ClientHint(ctx)
	}

	sub := ""
	if Resolver() == "subdomain" && ctx != nil {
		sub = SubdomainHint(ctx.Request().Host())
	}
	return MergeTenantHints(Resolver(), sub, clientHint, AllowHeaderFallback())
}

// MergeTenantHints is the pure resolver used by ResolveHint (unit-testable).
// resolver is "subdomain" or "header"; sub is Host-derived tenant when present.
func MergeTenantHints(resolver, sub, clientHint string, allowHeaderFallback bool) (string, error) {
	clientHint = strings.TrimSpace(clientHint)
	sub = strings.TrimSpace(sub)
	if resolver == "subdomain" {
		if sub != "" {
			if clientHint != "" && !strings.EqualFold(clientHint, sub) {
				return "", apperrors.ErrTenantHintConflict
			}
			return sub, nil
		}
		if !allowHeaderFallback {
			// Public: apex/reserved host must not accept client-supplied tenant.
			return "", nil
		}
	}
	return clientHint, nil
}

// SubdomainHint extracts tenant code from host like acme.example.com.
// When TENANCY_BASE_DOMAIN is set, only hosts under that apex are treated as
// built-in subdomains (so vanity hosts like crm.customer.com are not misread).
func SubdomainHint(host string) string {
	host = NormalizeHost(host)
	if host == "" {
		return ""
	}
	if base := BaseDomain(); base != "" {
		if !IsSubdomainOfBase(host) {
			return ""
		}
	}
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

// PaymentNotifyPath returns /api/payment/notify/{type}/{tenant} when tenancy is on.
func PaymentNotifyPath(tenantCode, notifyType string) string {
	notifyType = strings.ToLower(strings.TrimSpace(notifyType))
	tenantCode = strings.TrimSpace(tenantCode)
	if Enabled() && tenantCode != "" {
		return fmt.Sprintf("/api/payment/notify/%s/%s", notifyType, tenantCode)
	}
	return fmt.Sprintf("/api/payment/notify/%s", notifyType)
}
