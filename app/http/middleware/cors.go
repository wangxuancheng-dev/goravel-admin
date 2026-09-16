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

// Cors CORS 中间件，处理跨域请求
func Cors() http.Middleware {
	return newMiddleware("cors", func(ctx http.Context) {
		// 获取请求路径
		path := ctx.Request().Path()

		// 检查是否是 WebSocket 升级请求
		isWebSocket := strings.ToLower(ctx.Request().Header("Upgrade", "")) == "websocket" ||
			strings.ToLower(ctx.Request().Header("Connection", "")) == "upgrade"

		// 获取 CORS 配置的路径列表
		corsPaths := facades.Config().Get("cors.paths", []string{}).([]string)

		// 检查当前路径是否需要 CORS 处理
		needCors := false
		if len(corsPaths) == 0 {
			// 如果没有配置路径，默认对所有路径启用
			needCors = true
		} else {
			for _, corsPath := range corsPaths {
				// 支持通配符匹配
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

		// WebSocket 请求直接放行，不需要 CORS 处理（WebSocket 有自己的协议）
		if isWebSocket {
			ctx.Request().Next()
			return
		}

		if !needCors {
			ctx.Request().Next()
			return
		}

		// 获取 CORS 配置
		allowedOrigins := facades.Config().Get("cors.allowed_origins", []string{"*"}).([]string)
		allowedMethods := facades.Config().Get("cors.allowed_methods", []string{"*"}).([]string)
		allowedHeaders := facades.Config().Get("cors.allowed_headers", []string{"*"}).([]string)
		exposedHeaders := facades.Config().Get("cors.exposed_headers", []string{}).([]string)
		maxAge := facades.Config().GetInt("cors.max_age", 0)
		supportsCredentials := facades.Config().GetBool("cors.supports_credentials", false)

		// 获取请求的 Origin
		origin := ctx.Request().Header("Origin", "")

		// 检查是否允许该 Origin
		allowed := false
		var allowedOrigin string

		if len(allowedOrigins) > 0 && allowedOrigins[0] == "*" {
			// 允许所有源
			allowed = true
			allowedOrigin = "*"
		} else if origin != "" {
			// 检查是否在允许列表中
			if slices.Contains(allowedOrigins, origin) {
				allowed = true
				allowedOrigin = origin
			} else if host := originHost(origin); host != "" && services.NewTenantDomainService().IsActiveHost(host) {
				allowed = true
				allowedOrigin = origin
			}
		}

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

		// 对于非预检请求，设置 CORS 响应头
		if allowed && origin != "" {
			ctx.Response().Header("Access-Control-Allow-Origin", allowedOrigin)
			if supportsCredentials && allowedOrigin != "*" {
				ctx.Response().Header("Access-Control-Allow-Credentials", "true")
			}
		} else if len(allowedOrigins) > 0 && allowedOrigins[0] == "*" && origin != "" {
			// 如果配置允许所有源，且请求有 origin，设置 CORS 头
			ctx.Response().Header("Access-Control-Allow-Origin", "*")
		}

		// 设置暴露的响应头（非预检请求）
		if len(exposedHeaders) > 0 {
			ctx.Response().Header("Access-Control-Expose-Headers", strings.Join(exposedHeaders, ", "))
		}

		// 继续处理请求
		ctx.Request().Next()
	})
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
