package tenancy

import (
	"net"
	"net/url"
	"strings"

	"github.com/goravel/framework/facades"
)

// NormalizeHost lowercases host, strips scheme/path/port/trailing dot.
func NormalizeHost(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	raw = strings.ToLower(raw)
	if strings.Contains(raw, "://") {
		if u, err := url.Parse(raw); err == nil && u.Host != "" {
			raw = u.Host
		}
	}
	if i := strings.Index(raw, "/"); i >= 0 {
		raw = raw[:i]
	}
	raw = strings.TrimSuffix(raw, ".")
	if host, _, err := net.SplitHostPort(raw); err == nil {
		raw = host
	} else if i := strings.Index(raw, ":"); i >= 0 && !strings.Contains(raw, "]") {
		raw = raw[:i]
	}
	return strings.TrimSpace(raw)
}

// RequestHost prefers X-Forwarded-Host then Request.Host.
func RequestHost(hostHeader, forwardedHost string) string {
	host := strings.TrimSpace(forwardedHost)
	if host != "" && strings.Contains(host, ",") {
		host = strings.TrimSpace(strings.Split(host, ",")[0])
	}
	if host == "" {
		host = strings.TrimSpace(hostHeader)
	}
	return NormalizeHost(host)
}

// OriginHost extracts the host from a browser Origin header (https://acme.example.com).
func OriginHost(origin string) string {
	return NormalizeHost(origin)
}

// PickTenantHostCode resolves tenant code from request Host first, then Origin host.
// Origin is used for split SPA/API (Host=api.* while Origin=acme.* or a vanity domain).
func PickTenantHostCode(reqHost, origin string, resolve func(string) string) string {
	if resolve == nil {
		return ""
	}
	reqHost = NormalizeHost(reqHost)
	if code := resolve(reqHost); code != "" {
		return code
	}
	originHost := OriginHost(origin)
	if originHost != "" && originHost != reqHost {
		return resolve(originHost)
	}
	return ""
}

// BaseDomain returns TENANCY_BASE_DOMAIN (normalized).
func BaseDomain() string {
	return NormalizeHost(facades.Config().GetString("tenancy.base_domain", ""))
}

// DomainTarget returns TENANCY_DOMAIN_TARGET customers should CNAME to.
func DomainTarget() string {
	return NormalizeHost(facades.Config().GetString("tenancy.domain_target", ""))
}

// DomainVerifyPrefix returns DNS TXT owner label prefix.
func DomainVerifyPrefix() string {
	p := strings.TrimSpace(facades.Config().GetString("tenancy.domain_verify_prefix", "_goravel-tenant"))
	if p == "" {
		return "_goravel-tenant"
	}
	return strings.ToLower(p)
}

// IsSubdomainOfBase reports whether host is {label}.{base_domain}.
func IsSubdomainOfBase(host string) bool {
	host = NormalizeHost(host)
	base := BaseDomain()
	if host == "" || base == "" {
		return false
	}
	suffix := "." + base
	if !strings.HasSuffix(host, suffix) {
		return false
	}
	label := strings.TrimSuffix(host, suffix)
	return label != "" && !strings.Contains(label, ".")
}

// IsReservedOrPlatformHost rejects hosts that must never be tenant vanity domains.
func IsReservedOrPlatformHost(host string) bool {
	host = NormalizeHost(host)
	if host == "" || host == "localhost" || net.ParseIP(host) != nil {
		return true
	}
	base := BaseDomain()
	if base != "" {
		if host == base || host == "www."+base || host == "platform."+base || host == "api."+base {
			return true
		}
		// Built-in tenant subdomain space belongs to TENANCY_RESOLVER=subdomain, not vanity table.
		if IsSubdomainOfBase(host) {
			return true
		}
	}
	return false
}

// IsLockedPublicApexHost reports marketing/apex hosts where client-supplied
// tenant (Header/Query) must stay gated by TENANCY_ALLOW_HEADER_FALLBACK.
// Loopback, API hosts, and unknown hosts are not locked so public <img>
// ?tenant_code= works behind Vite changeOrigin / split SPA-API deploys.
func IsLockedPublicApexHost(host string) bool {
	host = NormalizeHost(host)
	if host == "" || host == "localhost" || net.ParseIP(host) != nil {
		return false
	}
	base := BaseDomain()
	if base == "" {
		return false
	}
	return host == base || host == "www."+base
}
