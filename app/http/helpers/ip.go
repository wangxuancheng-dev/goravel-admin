package helpers

import (
	"net"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/support/str"
)

// GetRealIP 获取客户端真实IP地址
// 优先从以下HTTP头获取（按顺序）：
// 1. CF-Connecting-IP (Cloudflare)
// 2. True-Client-IP
// 3. X-Real-IP
// 4. X-Forwarded-For (取第一个IP)
// 5. RemoteAddr
func GetRealIP(ctx http.Context) string {
	// 1. Cloudflare
	if ip := ctx.Request().Header("CF-Connecting-IP", ""); ip != "" {
		if parsedIP := parseIP(ip); parsedIP != "" {
			return parsedIP
		}
	}

	// 2. True-Client-IP
	if ip := ctx.Request().Header("True-Client-IP", ""); ip != "" {
		if parsedIP := parseIP(ip); parsedIP != "" {
			return parsedIP
		}
	}

	// 3. X-Real-IP
	if ip := ctx.Request().Header("X-Real-IP", ""); ip != "" {
		if parsedIP := parseIP(ip); parsedIP != "" {
			return parsedIP
		}
	}

	// 4. X-Forwarded-For (可能包含多个IP，取第一个)
	if forwardedFor := ctx.Request().Header("X-Forwarded-For", ""); !str.Of(forwardedFor).IsEmpty() {
		ips := str.Of(forwardedFor).Split(",")
		if len(ips) > 0 {
			ip := str.Of(ips[0]).Trim().String()
			if parsedIP := parseIP(ip); !str.Of(parsedIP).IsEmpty() {
				return parsedIP
			}
		}
	}

	// 5. RemoteAddr
	remoteAddr := ctx.Request().Ip()
	if parsedIP := parseIP(remoteAddr); parsedIP != "" {
		return parsedIP
	}

	return remoteAddr
}

// parseIP 解析并验证IP地址
func parseIP(ip string) string {
	ip = str.Of(ip).Trim().String()
	if str.Of(ip).IsEmpty() {
		return ""
	}

	// 如果包含端口，去掉端口
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}

	// 验证是否为有效IP
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return ""
	}

	// Prefer IPv4 form for IPv4-mapped addresses (:ffff:x.x.x.x).
	if v4 := parsedIP.To4(); v4 != nil {
		return v4.String()
	}
	// Localhost via IPv6 (::1) -> 127.0.0.1 so allow/blacklist tips and rules match common IPv4 entries.
	if parsedIP.IsLoopback() {
		return "127.0.0.1"
	}

	return parsedIP.String()
}
