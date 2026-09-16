package middleware

import (
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
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
//
// Framework gin Cors (rs/cors) is disabled via empty cors.paths so preflight is not
// aborted by the static allowlist before vanity hosts can be checked. This middleware
// wraps the gin ResponseWriter so ACAO is applied when the handler writes the body
// (set-before-Next alone is unreliable with Goravel Render).
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
		allowedMethods := configStringSlice("cors.allowed_methods", []string{"*"})
		allowedHeaders := configStringSlice("cors.allowed_headers", []string{"*"})
		exposedHeaders := configStringSlice("cors.exposed_headers", nil)
		maxAge := facades.Config().GetInt("cors.max_age", 0)
		supportsCredentials := facades.Config().GetBool("cors.supports_credentials", false)

		origin := ctx.Request().Header("Origin", "")
		allowed, allowedOrigin := resolveCorsOrigin(origin, allowedOrigins)

		if strings.EqualFold(ctx.Request().Method(), http.MethodOptions) {
			writeCorsPreflightHeaders(ctx, origin, allowed, allowedOrigin, allowedOrigins, allowedMethods, allowedHeaders, exposedHeaders, maxAge, supportsCredentials)
			ctx.Request().Abort(http.StatusNoContent)
			return
		}

		unwrap := wrapCorsWriter(ctx, origin, allowed, allowedOrigin, allowedOrigins, exposedHeaders, supportsCredentials)
		defer unwrap()
		ctx.Request().Next()
	})
}

// wrapCorsWriter injects CORS headers at WriteHeader/Write time (like rs/cors).
// Returns an unwrap func for defer.
func wrapCorsWriter(
	ctx http.Context,
	origin string,
	allowed bool,
	allowedOrigin string,
	allowedOrigins, exposedHeaders []string,
	supportsCredentials bool,
) func() {
	if origin == "" {
		return func() {}
	}
	acao := ""
	creds := false
	if allowed {
		acao = allowedOrigin
		creds = supportsCredentials && allowedOrigin != "*"
	} else if len(allowedOrigins) > 0 && allowedOrigins[0] == "*" {
		acao = "*"
	}
	if acao == "" {
		return func() {}
	}

	getter, ok := ctx.(ginInstance)
	if !ok {
		// Fallback: set headers before Next (best-effort without gin writer).
		writeCorsActualHeaders(ctx, origin, true, acao, allowedOrigins, exposedHeaders, creds)
		return func() {}
	}

	ginCtx := getter.Instance()
	orig := ginCtx.Writer
	w := &corsResponseWriter{
		ResponseWriter: orig,
		acao:           acao,
		credentials:    creds,
		exposed:        exposedHeaders,
	}
	ginCtx.Writer = w
	return func() {
		ginCtx.Writer = orig
	}
}

type corsResponseWriter struct {
	gin.ResponseWriter
	acao        string
	credentials bool
	exposed     []string
	applied     bool
}

func (w *corsResponseWriter) applyHeaders() {
	if w.applied || w.acao == "" {
		return
	}
	w.applied = true
	h := w.ResponseWriter.Header()
	h.Set("Access-Control-Allow-Origin", w.acao)
	if w.credentials {
		h.Set("Access-Control-Allow-Credentials", "true")
	}
	if len(w.exposed) > 0 {
		h.Set("Access-Control-Expose-Headers", strings.Join(w.exposed, ", "))
	}
	h.Add("Vary", "Origin")
}

func (w *corsResponseWriter) WriteHeader(code int) {
	w.applyHeaders()
	w.ResponseWriter.WriteHeader(code)
}

func (w *corsResponseWriter) Write(b []byte) (int, error) {
	w.applyHeaders()
	return w.ResponseWriter.Write(b)
}

func (w *corsResponseWriter) WriteString(s string) (int, error) {
	w.applyHeaders()
	return w.ResponseWriter.WriteString(s)
}

func writeCorsActualHeaders(
	ctx http.Context,
	origin string,
	allowed bool,
	allowedOrigin string,
	allowedOrigins, exposedHeaders []string,
	supportsCredentials bool,
) {
	if origin == "" {
		return
	}
	response := ctx.Response()
	if allowed {
		response.Header("Access-Control-Allow-Origin", allowedOrigin)
		if supportsCredentials && allowedOrigin != "*" {
			response.Header("Access-Control-Allow-Credentials", "true")
		}
	} else if len(allowedOrigins) > 0 && allowedOrigins[0] == "*" {
		response.Header("Access-Control-Allow-Origin", "*")
	}
	if len(exposedHeaders) > 0 {
		response.Header("Access-Control-Expose-Headers", strings.Join(exposedHeaders, ", "))
	}
}

// WriteCorsPreflightResponse applies CORS headers for an OPTIONS request (middleware or Fallback).
func WriteCorsPreflightResponse(ctx http.Context) {
	allowedOrigins := configStringSlice("cors.allowed_origins", []string{"*"})
	allowedMethods := configStringSlice("cors.allowed_methods", []string{"*"})
	allowedHeaders := configStringSlice("cors.allowed_headers", []string{"*"})
	exposedHeaders := configStringSlice("cors.exposed_headers", nil)
	maxAge := facades.Config().GetInt("cors.max_age", 0)
	supportsCredentials := facades.Config().GetBool("cors.supports_credentials", false)
	origin := ctx.Request().Header("Origin", "")
	allowed, allowedOrigin := resolveCorsOrigin(origin, allowedOrigins)
	writeCorsPreflightHeaders(ctx, origin, allowed, allowedOrigin, allowedOrigins, allowedMethods, allowedHeaders, exposedHeaders, maxAge, supportsCredentials)
}

func writeCorsPreflightHeaders(
	ctx http.Context,
	origin string,
	allowed bool,
	allowedOrigin string,
	allowedOrigins, allowedMethods, allowedHeaders, exposedHeaders []string,
	maxAge int,
	supportsCredentials bool,
) {
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

	response.Header("Vary", "Origin")
}

// IsCorsOriginAllowed reports whether Origin passes the same rules as Cors middleware.
func IsCorsOriginAllowed(origin string) bool {
	ok, _ := resolveCorsOrigin(origin, configStringSlice("cors.allowed_origins", []string{"*"}))
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
