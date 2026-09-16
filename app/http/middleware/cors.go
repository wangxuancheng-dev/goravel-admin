package middleware

import (
	"net/url"
	"slices"
	"strings"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"
	"github.com/rs/cors"

	"goravel/app/services"
	"goravel/app/tenancy"
)

// Cors CORS middleware for cross-origin requests.
// Allowed origins:
//  1. CORS_ALLOWED_ORIGINS exact match or wildcard host patterns (e.g. https://*.example.com)
//  2. Any host under TENANCY_BASE_DOMAIN (tenant subdomains)
//  3. Active vanity domains in tenant_domains (status=active)
//
// Framework gin Cors is disabled (empty cors.paths). This middleware uses rs/cors with
// AllowOriginFunc so vanity hosts are allowed on both OPTIONS and real responses
// (ResponseWriter wrapping), which plain Header()-before-Next cannot do reliably.
func Cors() http.Middleware {
	return newMiddleware("cors", func(ctx http.Context) {
		path := ctx.Request().Path()

		isWebSocket := strings.ToLower(ctx.Request().Header("Upgrade", "")) == "websocket" ||
			strings.ToLower(ctx.Request().Header("Connection", "")) == "upgrade"

		corsPaths := configStringSlice("cors.paths", nil)

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

		if isWebSocket {
			ctx.Request().Next()
			return
		}

		if !needCors {
			ctx.Request().Next()
			return
		}

		allowedOrigins := configStringSlice("cors.allowed_origins", []string{"*"})
		allowedMethods := configStringSlice("cors.allowed_methods", []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"})
		allowedHeaders := configStringSlice("cors.allowed_headers", []string{"*"})
		exposedHeaders := configStringSlice("cors.exposed_headers", nil)
		maxAge := facades.Config().GetInt("cors.max_age", 0)
		supportsCredentials := facades.Config().GetBool("cors.supports_credentials", false)

		if len(allowedMethods) == 1 && allowedMethods[0] == "*" {
			allowedMethods = []string{
				http.MethodGet, http.MethodPost, http.MethodHead,
				http.MethodPut, http.MethodDelete, http.MethodPatch, http.MethodOptions,
			}
		}

		opts := cors.Options{
			AllowedMethods:      allowedMethods,
			AllowedHeaders:      allowedHeaders,
			ExposedHeaders:      exposedHeaders,
			MaxAge:              maxAge,
			AllowCredentials:    supportsCredentials,
			AllowPrivateNetwork: true,
		}
		if len(allowedOrigins) > 0 && allowedOrigins[0] == "*" {
			opts.AllowedOrigins = []string{"*"}
		} else {
			origins := allowedOrigins
			opts.AllowOriginFunc = func(origin string) bool {
				ok, _ := resolveCorsOrigin(origin, origins)
				return ok
			}
		}

		cors.New(opts).HandlerFunc(ctx.Response().Writer(), ctx.Request().Origin())

		// Match framework gin Cors: abort preflight so route Fallback is not hit.
		if strings.EqualFold(ctx.Request().Method(), http.MethodOptions) &&
			ctx.Request().Header("Access-Control-Request-Method", "") != "" {
			ctx.Request().Abort(http.StatusNoContent)
			return
		}

		ctx.Request().Next()
	})
}

// WriteCorsPreflightResponse applies CORS for OPTIONS when Fallback is reached.
func WriteCorsPreflightResponse(ctx http.Context) {
	allowedOrigins := configStringSlice("cors.allowed_origins", []string{"*"})
	allowedMethods := configStringSlice("cors.allowed_methods", []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"})
	allowedHeaders := configStringSlice("cors.allowed_headers", []string{"*"})
	exposedHeaders := configStringSlice("cors.exposed_headers", nil)
	maxAge := facades.Config().GetInt("cors.max_age", 0)
	supportsCredentials := facades.Config().GetBool("cors.supports_credentials", false)

	if len(allowedMethods) == 1 && allowedMethods[0] == "*" {
		allowedMethods = []string{
			http.MethodGet, http.MethodPost, http.MethodHead,
			http.MethodPut, http.MethodDelete, http.MethodPatch, http.MethodOptions,
		}
	}

	opts := cors.Options{
		AllowedMethods:      allowedMethods,
		AllowedHeaders:      allowedHeaders,
		ExposedHeaders:      exposedHeaders,
		MaxAge:              maxAge,
		AllowCredentials:    supportsCredentials,
		AllowPrivateNetwork: true,
	}
	if len(allowedOrigins) > 0 && allowedOrigins[0] == "*" {
		opts.AllowedOrigins = []string{"*"}
	} else {
		origins := allowedOrigins
		opts.AllowOriginFunc = func(origin string) bool {
			ok, _ := resolveCorsOrigin(origin, origins)
			return ok
		}
	}
	cors.New(opts).HandlerFunc(ctx.Response().Writer(), ctx.Request().Origin())
}

// IsCorsOriginAllowed reports whether Origin passes the same rules as Cors middleware.
func IsCorsOriginAllowed(origin string) bool {
	ok, _ := resolveCorsOrigin(origin, configStringSlice("cors.allowed_origins", []string{"*"}))
	return ok
}

// resolveCorsOrigin returns whether Origin is allowed and the value for ACAO.
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

func isCorsBaseDomainHost(host string) bool {
	host = tenancy.NormalizeHost(host)
	base := tenancy.BaseDomain()
	if host == "" || base == "" {
		return false
	}
	return host == base || strings.HasSuffix(host, "."+base)
}

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

func configStringSlice(key string, fallback []string) []string {
	value := facades.Config().Get(key, fallback)
	switch v := value.(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return fallback
	}
}
