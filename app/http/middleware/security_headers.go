package middleware

import (
	"fmt"
	"strings"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"
)

// SecurityHeaders adds baseline browser security headers for HTML/API responses.
func SecurityHeaders() http.Middleware {
	return newMiddleware("security_headers", func(ctx http.Context) {
		path := ctx.Request().Path()
		isWebSocket := strings.EqualFold(ctx.Request().Header("Upgrade", ""), "websocket")

		ctx.Response().Header("X-Content-Type-Options", "nosniff")
		ctx.Response().Header("X-Frame-Options", "SAMEORIGIN")
		ctx.Response().Header("Referrer-Policy", "strict-origin-when-cross-origin")
		ctx.Response().Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		// HSTS only when explicitly configured (set behind HTTPS terminators).
		if maxAge := facades.Config().GetInt("http.security.hsts_max_age", 0); maxAge > 0 {
			ctx.Response().Header("Strict-Transport-Security", fmt.Sprintf("max-age=%d; includeSubDomains", maxAge))
		}

		// Avoid breaking SPA assets / websocket upgrades with a strict CSP.
		if !isWebSocket && !strings.HasPrefix(path, "/api/") {
			ctx.Response().Header("Content-Security-Policy", "frame-ancestors 'self'")
		}

		ctx.Request().Next()
	})
}
