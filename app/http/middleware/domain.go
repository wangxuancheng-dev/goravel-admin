package middleware

import (
	"net"
	"strings"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"

	"goravel/app/http/response"
	"goravel/app/services"
)

// Domain host allowlist middleware (platform console; does not auto-allow tenant vanity hosts).
func Domain(configValueOrDomains ...any) http.Middleware {
	return domainMiddleware(false, configValueOrDomains...)
}

// DomainAllowTenantVanity allows DOMAINS_ADMIN plus active tenant_domains hosts (admin API).
func DomainAllowTenantVanity(configValueOrDomains ...any) http.Middleware {
	return domainMiddleware(true, configValueOrDomains...)
}

func domainMiddleware(allowTenantVanity bool, configValueOrDomains ...any) http.Middleware {
	name := "domain"
	if allowTenantVanity {
		name = "domain_tenant_vanity"
	}
	return newMiddleware(name, func(ctx http.Context) {
		var domains []string

		if len(configValueOrDomains) == 0 {
			ctx.Request().Next()
			return
		}

		parseConfigValue := func(value any) []string {
			if value == nil {
				return nil
			}

			switch v := value.(type) {
			case []string:
				return v
			case string:
				if v == "" {
					return nil
				}
				domainsList := strings.Split(v, ",")
				result := make([]string, 0, len(domainsList))
				for _, d := range domainsList {
					d = strings.TrimSpace(d)
					if d != "" {
						result = append(result, d)
					}
				}
				return result
			case []any:
				result := make([]string, 0, len(v))
				for _, item := range v {
					if str, ok := item.(string); ok && str != "" {
						result = append(result, str)
					}
				}
				return result
			default:
				return nil
			}
		}

		for _, param := range configValueOrDomains {
			if param == nil {
				continue
			}

			switch v := param.(type) {
			case []string:
				if len(v) > 0 {
					domains = append(domains, v...)
				}
			case string:
				if v != "" {
					configValue := facades.Config().Get(v, nil)
					if configValue != nil {
						parsedDomains := parseConfigValue(configValue)
						if len(parsedDomains) > 0 {
							domains = append(domains, parsedDomains...)
							continue
						}
					}
					domains = append(domains, v)
				}
			case []any:
				for _, item := range v {
					if str, ok := item.(string); ok && str != "" {
						domains = append(domains, str)
					}
				}
			default:
				parsedDomains := parseConfigValue(v)
				if len(parsedDomains) > 0 {
					domains = append(domains, parsedDomains...)
				}
			}
		}

		if len(domains) == 0 {
			ctx.Request().Next()
			return
		}

		host := ctx.Request().Header("X-Forwarded-Host", "")
		if host == "" {
			host = ctx.Request().Host()
		}

		if host != "" && strings.Contains(host, ",") {
			host = strings.TrimSpace(strings.Split(host, ",")[0])
		}

		if facades.Config().GetBool("app.debug", false) {
			facades.Log().Debugf("Domain middleware: Host detection - X-Forwarded-Host: %s, Host(): %s, Final host: %s",
				ctx.Request().Header("X-Forwarded-Host", ""),
				ctx.Request().Host(),
				host)
		}

		normalizedHost := normalizeHost(host)

		if facades.Config().GetBool("app.debug", false) {
			facades.Log().Debugf("Domain middleware: Normalized host: %s, Configured domains: %v", normalizedHost, domains)
		}

		allowed := false
		var matchedDomain string
		for _, allowedDomain := range domains {
			normalizedAllowed := normalizeHost(allowedDomain)
			if normalizedHost == normalizedAllowed || matchDomain(normalizedHost, normalizedAllowed) {
				allowed = true
				matchedDomain = allowedDomain
				break
			}
		}

		if !allowed && allowTenantVanity {
			if services.NewTenantDomainService().IsActiveHost(normalizedHost) {
				allowed = true
				matchedDomain = normalizedHost
			}
		}

		if !allowed {
			response.Abort(ctx, http.StatusForbidden, "domain_not_allowed")
			return
		}

		if facades.Config().GetBool("app.debug", false) {
			facades.Log().Debugf("Domain middleware: Access allowed. Request host: %s, Matched domain: %s", normalizedHost, matchedDomain)
		}

		ctx.Request().Next()
	})
}

func normalizeHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return ""
	}

	if hostname, _, err := net.SplitHostPort(host); err == nil {
		host = hostname
	}

	return host
}

func matchDomain(host, pattern string) bool {
	if pattern == "" {
		return false
	}

	if after, ok := strings.CutPrefix(pattern, "*."); ok {
		suffix := after
		if strings.HasSuffix(host, "."+suffix) || host == suffix {
			return true
		}
	}

	if before, ok := strings.CutSuffix(pattern, ".*"); ok {
		prefix := before
		if strings.HasPrefix(host, prefix+".") || host == prefix {
			return true
		}
	}

	return false
}