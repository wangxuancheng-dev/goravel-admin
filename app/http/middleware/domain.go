package middleware

import (
	"net"
	"strings"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"

	"goravel/app/http/response"
)

// Domain 域名验证中间件
func Domain(configValueOrDomains ...any) http.Middleware {
	return newMiddleware("domain", func(ctx http.Context) {
		var domains []string

		// 如果没有参数，不验证（允许所有域名）
		if len(configValueOrDomains) == 0 {
			ctx.Request().Next()
			return
		}

		// 解析配置值的辅助函数
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
				// 分割逗号分隔的域名
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

		// 处理参数
		for _, param := range configValueOrDomains {
			if param == nil {
				continue
			}

			switch v := param.(type) {
			case []string:
				// 如果是字符串数组，直接使用
				if len(v) > 0 {
					domains = append(domains, v...)
				}
			case string:
				// 如果是字符串，可能是单个域名或配置键
				if v != "" {
					// 先尝试作为配置键读取
					configValue := facades.Config().Get(v, nil)
					if configValue != nil {
						// 如果配置存在，解析配置值
						parsedDomains := parseConfigValue(configValue)
						if len(parsedDomains) > 0 {
							domains = append(domains, parsedDomains...)
							continue
						}
					}
					// 否则当作域名
					domains = append(domains, v)
				}
			case []any:
				// 如果是 any 数组，递归处理
				for _, item := range v {
					if str, ok := item.(string); ok && str != "" {
						domains = append(domains, str)
					}
				}
			default:
				// 其他类型，尝试解析为配置值
				parsedDomains := parseConfigValue(v)
				if len(parsedDomains) > 0 {
					domains = append(domains, parsedDomains...)
				}
			}
		}

		// 如果没有指定允许的域名，允许所有域名访问（直接放行）
		if len(domains) == 0 {
			ctx.Request().Next()
			return
		}

		// 获取请求的 Host
		// 优先从 X-Forwarded-Host 获取（适用于反向代理场景）
		host := ctx.Request().Header("X-Forwarded-Host", "")
		if host == "" {
			// 使用框架提供的 Host() 方法获取（推荐方式）
			host = ctx.Request().Host()
		}

		// 如果 X-Forwarded-Host 包含多个值（逗号分隔），取第一个
		if host != "" && strings.Contains(host, ",") {
			host = strings.TrimSpace(strings.Split(host, ",")[0])
		}

		// 调试日志：记录获取到的 Host 值
		if facades.Config().GetBool("app.debug", false) {
			facades.Log().Debugf("Domain middleware: Host detection - X-Forwarded-Host: %s, Host(): %s, Final host: %s",
				ctx.Request().Header("X-Forwarded-Host", ""),
				ctx.Request().Host(),
				host)
		}

		// 规范化 Host（移除端口号，转换为小写）
		normalizedHost := normalizeHost(host)

		// 调试日志：记录规范化后的 Host 和配置的域名
		if facades.Config().GetBool("app.debug", false) {
			facades.Log().Debugf("Domain middleware: Normalized host: %s, Configured domains: %v", normalizedHost, domains)
		}

		// 检查是否在允许的域名列表中
		allowed := false
		var matchedDomain string
		for _, allowedDomain := range domains {
			normalizedAllowed := normalizeHost(allowedDomain)
			// 支持精确匹配和通配符匹配
			if normalizedHost == normalizedAllowed || matchDomain(normalizedHost, normalizedAllowed) {
				allowed = true
				matchedDomain = allowedDomain
				break
			}
		}

		if !allowed {
			// 域名不在允许列表中，拒绝访问
			response.Abort(ctx, http.StatusForbidden, "domain_not_allowed")
			return
		}

		// 记录匹配的域名（仅在调试模式下）
		if facades.Config().GetBool("app.debug", false) {
			facades.Log().Debugf("Domain middleware: Access allowed. Request host: %s, Matched domain: %s", normalizedHost, matchedDomain)
		}

		// 域名验证通过，继续处理请求
		ctx.Request().Next()
	})
}

// normalizeHost 规范化域名（移除端口号，转换为小写，去除前后空格）
func normalizeHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return ""
	}

	// 移除端口号
	if hostname, _, err := net.SplitHostPort(host); err == nil {
		host = hostname
	}

	return host
}

// matchDomain 域名匹配，支持通配符
// 例如：*.example.com 可以匹配 a.example.com, b.example.com 等
func matchDomain(host, pattern string) bool {
	if pattern == "" {
		return false
	}

	// 如果模式以 * 开头，进行通配符匹配
	if after, ok := strings.CutPrefix(pattern, "*."); ok {
		// 移除 *.
		suffix := after
		// 检查 host 是否以 .suffix 结尾
		if strings.HasSuffix(host, "."+suffix) || host == suffix {
			return true
		}
	}

	// 如果模式以 * 结尾，进行前缀匹配
	if before, ok := strings.CutSuffix(pattern, ".*"); ok {
		prefix := before
		if strings.HasPrefix(host, prefix+".") || host == prefix {
			return true
		}
	}

	return false
}
