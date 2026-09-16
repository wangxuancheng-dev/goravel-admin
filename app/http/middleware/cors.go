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

		// 处理预检请求 (OPTIONS) - 必须在设置其他头之前处理
		if ctx.Request().Method() == "OPTIONS" {
			// 对于预检请求，必须设置 CORS 头
			// 即使 origin 不在允许列表中，也要返回 CORS 头（只是不设置 Access-Control-Allow-Origin）
			// 这样浏览器才能正确判断，而不是因为状态码问题而失败

			// 创建响应对象并设置所有 CORS 头
			response := ctx.Response()

			if allowed && origin != "" {
				// Origin 在允许列表中
				response.Header("Access-Control-Allow-Origin", allowedOrigin)
				if supportsCredentials && allowedOrigin != "*" {
					response.Header("Access-Control-Allow-Credentials", "true")
				}
			} else if len(allowedOrigins) > 0 && allowedOrigins[0] == "*" {
				// 配置允许所有源
				response.Header("Access-Control-Allow-Origin", "*")
			} else if origin != "" {
				// Origin 不在允许列表中，不设置 Access-Control-Allow-Origin
				// 浏览器会拒绝请求，但至少不会因为状态码问题而失败
			}

			// 设置允许的方法（对于预检请求，这些头必须设置）
			methodsStr := "*"
			if len(allowedMethods) > 0 && allowedMethods[0] != "*" {
				methodsStr = strings.Join(allowedMethods, ", ")
			}
			response.Header("Access-Control-Allow-Methods", methodsStr)

			// 设置允许的请求头
			headersStr := "*"
			if len(allowedHeaders) > 0 && allowedHeaders[0] != "*" {
				headersStr = strings.Join(allowedHeaders, ", ")
			}
			response.Header("Access-Control-Allow-Headers", headersStr)

			// 设置暴露的响应头
			if len(exposedHeaders) > 0 {
				response.Header("Access-Control-Expose-Headers", strings.Join(exposedHeaders, ", "))
			}

			// 设置预检请求的缓存时间
			if maxAge > 0 {
				response.Header("Access-Control-Max-Age", strconv.Itoa(maxAge))
			}

			// 返回 204 No Content 并终止请求处理
			// 对于 OPTIONS 请求，返回空响应体，状态码为 204
			// 使用 Json 方法返回空对象，然后调用 Abort() 终止请求
			_ = response.Json(http.StatusNoContent, http.Json{}).Abort()
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
