package config

import (
	"strings"

	"github.com/goravel/framework/facades"
)

func init() {
	config := facades.Config()

	// 从环境变量读取允许的域名，支持逗号分隔的多个域名
	// 例如: CORS_ALLOWED_ORIGINS=https://admin.example.com,http://localhost:3007
	corsOriginsEnv := config.Env("CORS_ALLOWED_ORIGINS", "*")
	var allowedOrigins []string
	corsOriginsStr := ""
	if str, ok := corsOriginsEnv.(string); ok {
		corsOriginsStr = str
	} else {
		corsOriginsStr = "*"
	}

	if corsOriginsStr == "*" {
		allowedOrigins = []string{"*"}
	} else {
		// 分割逗号分隔的域名，并去除空格
		origins := strings.Split(corsOriginsStr, ",")
		allowedOrigins = make([]string, 0, len(origins))
		for _, origin := range origins {
			origin = strings.TrimSpace(origin)
			if origin != "" {
				allowedOrigins = append(allowedOrigins, origin)
			}
		}
		// 如果没有有效的域名，默认允许所有
		if len(allowedOrigins) == 0 {
			allowedOrigins = []string{"*"}
		}
	}

	// 从环境变量读取允许的方法，支持逗号分隔
	corsMethodsEnv := config.Env("CORS_ALLOWED_METHODS", "GET,POST,PUT,DELETE,PATCH,OPTIONS")
	var allowedMethods []string
	corsMethodsStr := ""
	if str, ok := corsMethodsEnv.(string); ok {
		corsMethodsStr = str
	} else {
		corsMethodsStr = "GET,POST,PUT,DELETE,PATCH,OPTIONS"
	}

	if corsMethodsStr == "*" {
		allowedMethods = []string{"*"}
	} else {
		methods := strings.Split(corsMethodsStr, ",")
		allowedMethods = make([]string, 0, len(methods))
		for _, method := range methods {
			method = strings.TrimSpace(strings.ToUpper(method))
			if method != "" {
				allowedMethods = append(allowedMethods, method)
			}
		}
		if len(allowedMethods) == 0 {
			allowedMethods = []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"}
		}
	}

	// 从环境变量读取允许的请求头，支持逗号分隔
	corsHeadersEnv := config.Env("CORS_ALLOWED_HEADERS", "Content-Type,Authorization,X-Requested-With,X-Timezone,Accept,Origin,Accept-Language,X-Tenant-ID")
	var allowedHeaders []string
	corsHeadersStr := ""
	if str, ok := corsHeadersEnv.(string); ok {
		corsHeadersStr = str
	} else {
		corsHeadersStr = "Content-Type,Authorization,X-Requested-With,X-Timezone,Accept,Origin,X-Tenant-ID"
	}

	if corsHeadersStr == "*" {
		allowedHeaders = []string{"*"}
	} else {
		headers := strings.Split(corsHeadersStr, ",")
		allowedHeaders = make([]string, 0, len(headers))
		for _, header := range headers {
			header = strings.TrimSpace(header)
			if header != "" {
				allowedHeaders = append(allowedHeaders, header)
			}
		}
		if len(allowedHeaders) == 0 {
			allowedHeaders = []string{"*"}
		}
	}

	// 从环境变量读取暴露的响应头，支持逗号分隔
	corsExposedHeadersEnv := config.Env("CORS_EXPOSED_HEADERS", "Authorization,X-Trace-Id")
	var exposedHeaders []string
	corsExposedHeadersStr := ""
	if str, ok := corsExposedHeadersEnv.(string); ok {
		corsExposedHeadersStr = str
	} else {
		corsExposedHeadersStr = "Authorization,X-Trace-Id"
	}

	if corsExposedHeadersStr != "" {
		headers := strings.Split(corsExposedHeadersStr, ",")
		exposedHeaders = make([]string, 0, len(headers))
		for _, header := range headers {
			header = strings.TrimSpace(header)
			if header != "" {
				exposedHeaders = append(exposedHeaders, header)
			}
		}
	}

	config.Add("cors", map[string]any{
		// Cross-Origin Resource Sharing (CORS) Configuration
		//
		// Here you may configure your settings for cross-origin resource sharing
		// or "CORS". This determines what cross-origin operations may execute
		// in web browsers. You are free to adjust these settings as needed.
		//
		// To learn more: https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS
		//
		// Environment Variables:
		//   CORS_ALLOWED_ORIGINS: 允许的域名，多个用逗号分隔，例如: https://admin.example.com,http://localhost:3007
		//   CORS_ALLOWED_METHODS: 允许的HTTP方法，多个用逗号分隔，默认: GET,POST,PUT,DELETE,PATCH,OPTIONS
		//   CORS_ALLOWED_HEADERS: 允许的请求头，多个用逗号分隔
		//   CORS_EXPOSED_HEADERS: 暴露的响应头，多个用逗号分隔
		//   CORS_MAX_AGE: 预检请求缓存时间（秒），默认: 3600
		//   CORS_SUPPORTS_CREDENTIALS: 是否支持凭证，默认: true
		// "paths":                []string{"api/*"}, // 只对 API 路由启用 CORS
		"paths":                []string{}, // 空数组表示对所有路由启用 CORS
		"allowed_methods":      allowedMethods,
		"allowed_origins":      allowedOrigins,
		"allowed_headers":      allowedHeaders,
		"exposed_headers":      exposedHeaders,
		"max_age":              config.Env("CORS_MAX_AGE", 3600),
		"supports_credentials": config.Env("CORS_SUPPORTS_CREDENTIALS", true),
	})
}
