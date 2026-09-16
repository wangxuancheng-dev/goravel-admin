package middleware

import (
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"

	"goravel/app/services"
	"goravel/app/tenancy"
)

// Cors CORS middleware for cross-origin requests.
// Allowed origins:
//  1. CORS_ALLOWED_ORIGINS exact match or wildcard host patterns (e.g. https://*.example.com)
//  2. Any host under TENANCY_BASE_DOMAIN (tenant subdomains)
//  3. Active vanity domains in tenant_domains (status=active)
func Cors() http.Middleware {
	return newMiddleware("cors", func(ctx http.Context) {
		path := ctx.Request().Path()

		isWebSocket := strings.ToLower(ctx.Request().Header("Upgrade", "")) == "websocket" ||
			strings.ToLower(ctx.Request().Header("Connection", "")) == "upgrade"

		corsPaths := facades.Config().Get("cors.paths", []string{}).([]string)

		needCors := false
		if len(corsPaths) == 0 {
			needCors = true
		} else {
			for _, corsPath := range corsPaths {
				if strings.HasSuffix(corsPath, "*") {
					prefix := strings.TrimSuffix(corsPath, "*")
					if strings.HasPrefix(path, prefix) {
						needCors = true
						break
					}
				} else if path == corsPath {
					needCors = true
					break
				}
			}
		}

		// WebSocket has its own handshake; skip HTTP CORS headers here.
		if isWebSocket {
			ctx.Request().Next()
			return
		}

		if !needCors {
			ctx.Request().Next()
			return
		}

		allowedOrigins := facades.Config().Get("cors.allowed_origins", []string{"*"}).([]string)
		allowedMethods := facades.Config().Get("cors.allowed_methods", []string{"*"}).([]string)
		allowedHeaders := facades.Config().Get("cors.allowed_headers", []string{"*"}).([]string)
		exposedHeaders := facades.Config().Get("cors.exposed_headers", []string{}).([]string)
		maxAge := facades.Config().GetInt("cors.max_age", 0)
		supportsCredentials := facades.Config().GetBool("cors.supports_credentials", false)

		origin := ctx.Request().Header("Origin", "")
		allowed, allowedOrigin := resolveCorsOrigin(origin, allowedOrigins)

		// Preflight: set CORS headers then Abort with 204 (no body).
		// Do not use Json(204) — gin NoRoute/fallback can still run if the chain is not aborted cleanly.
		// Do not call Next() after Abort.
		if ctx.Request().Method() == http.MethodOptions {
			response := ctx.Response()

			if allowed && origin != "" {
				response.Header("Access-Control-Allow-Origin", allowedOrigin)
				if supportsCredentials && allowedOrigin != "*" {
					response.Header("Access-Control-Allow-Credentials", "true")
				}
			} else if len(allowedOrigins) > 0 && allowedOrigins[0] == "*" {
				response.Header("Access-Control-Allow-Origin", "*")
			}

			methodsStr := "*"
			if len(allowedMethods) > 0 && allowedMethods[0] != "*" {
				methodsStr = strings.Join(allowedMethods, ", ")
			}
			response.Header("Access-Control-Allow-Methods", methodsStr)

			headersStr := "*"
			if len(allowedHeaders) > 0 && allowedHeaders[0] != "*" {
				headersStr = strings.Join(allowedHeaders, ", ")
			}
			response.Header("Access-Control-Allow-Headers", headersStr)

			if len(exposedHeaders) > 0 {
				response.Header("Access-Control-Expose-Headers", strings.Join(exposedHeaders, ", "))
			}

			if maxAge > 0 {
				response.Header("Access-Control-Max-Age", strconv.Itoa(maxAge))
			}

			// NoContent.Abort -> gin AbortWithStatus(204); do not call Next().
			_ = response.NoContent(http.StatusNoContent).Abort()
			return
		}

		if allowed && origin != "" {
			ctx.Response().Header("Access-Control-Allow-Origin", allowedOrigin)
			if supportsCredentials && allowedOrigin != "*" {
				ctx.Response().Header("Access-Control-Allow-Credentials", "true")
			}
		} else if len(allowedOrigins) > 0 && allowedOrigins[0] == "*" && origin != "" {
			ctx.Response().Header("Access-Control-Allow-Origin", "*")
		}

		if len(exposedHeaders) > 0 {
			ctx.Response().Header("Access-Control-Expose-Headers", strings.Join(exposedHeaders, ", "))
		}

		ctx.Request().Next()
	})
}

// IsCorsOriginAllowed reports whether Origin passes the same rules as Cors middleware.
func IsCorsOriginAllowed(origin string) bool {
	allowedOrigins, _ := facades.Config().Get("cors.allowed_origins", []string{"*"}).([]string)
	ok, _ := resolveCorsOrigin(origin, allowedOrigins)
	return ok
}

// resolveCorsOrigin returns whether Origin is allowed and the value for ACAO.
// Dynamic matches (subdomain / vanity / wildcard) always echo the request Origin
// so credentials mode works.
func resolveCorsOrigin(origin string, allowedOrigins []string) (bool, string) {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return false, ""
	}

	if len(allowedOrigins) > 0 && allowedOrigins[0] == "*" {
		return true, "*"
	}

	if slices.Contains(allowedOrigins, origin) {
		return true, origin
	}

	if matchCorsOriginPattern(origin, allowedOrigins) {
		return true, origin
	}

	host := originHost(origin)
	if host == "" {
		return false, ""
	}

	if isCorsBaseDomainHost(host) {
		return true, origin
	}

	if services.NewTenantDomainService().IsActiveHost(host) {
		return true, origin
	}

	return false, ""
}

// isCorsBaseDomainHost allows apex and any subdomain of TENANCY_BASE_DOMAIN.
func isCorsBaseDomainHost(host string) bool {
	host = tenancy.NormalizeHost(host)
	base := tenancy.BaseDomain()
	if host == "" || base == "" {
		return false
	}
	return host == base || strings.HasSuffix(host, "."+base)
}

// matchCorsOriginPattern supports wildcard entries like https://*.example.com or *.example.com.
// Exact origins (no *) are handled by slices.Contains in resolveCorsOrigin.
func matchCorsOriginPattern(origin string, patterns []string) bool {
	host := originHost(origin)
	if host == "" {
		return false
	}
	ou, err := url.Parse(origin)
	if err != nil {
		return false
	}

	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "" || p == "*" || !strings.Contains(p, "*") {
			continue
		}
		if strings.Contains(p, "://") {
			pu, err := url.Parse(p)
			if err != nil || pu.Host == "" {
				continue
			}
			if pu.Scheme != "" && ou.Scheme != "" && !strings.EqualFold(pu.Scheme, ou.Scheme) {
				continue
			}
			patternHost := tenancy.NormalizeHost(pu.Host)
			if host == patternHost || matchDomain(host, patternHost) {
				return true
			}
			continue
		}
		patternHost := tenancy.NormalizeHost(p)
		if host == patternHost || matchDomain(host, patternHost) {
			return true
		}
	}
	return false
}

func originHost(origin string) string {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return ""
	}
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return tenancy.NormalizeHost(origin)
	}
	return tenancy.NormalizeHost(u.Host)
}
