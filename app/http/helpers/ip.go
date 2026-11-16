package helpers

import (
	"strings"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"
	"github.com/goravel/framework/support/str"

	"goravel/app/support/proxies"
)

// GetRealIP 获取客户端真实IP地址。
//
// 代理头可被客户端任意伪造，只有直连对端（TCP RemoteAddr）属于可信代理
// （http.trusted_proxies，默认回环 + 内网段）时才读取：
//  1. http.trusted_ip_header 指定的头（如 CF-Connecting-IP，需确保源站只能经由该 CDN 访问）
//  2. X-Forwarded-For：从右往左跳过可信代理，取第一个不可信地址（左侧由客户端控制）
//  3. X-Real-IP
//
// 否则直接返回 RemoteAddr。限流、登录锁定、IP 白/黑名单都依赖此结果。
func GetRealIP(ctx http.Context) string {
	peer := peerIP(ctx)
	trusted := proxies.FromConfig()
	if !trusted.Contains(peer) {
		return peer
	}

	if header := strings.TrimSpace(facades.Config().GetString("http.trusted_ip_header", "")); header != "" {
		if parsedIP := proxies.ParseIP(ctx.Request().Header(header, "")); parsedIP != "" {
			return parsedIP
		}
	}

	if forwardedFor := ctx.Request().Header("X-Forwarded-For", ""); !str.Of(forwardedFor).IsEmpty() {
		if ip := clientFromForwardedFor(forwardedFor, trusted); ip != "" {
			return ip
		}
	}

	if parsedIP := proxies.ParseIP(ctx.Request().Header("X-Real-IP", "")); parsedIP != "" {
		return parsedIP
	}

	return peer
}

// TrustedForwardedHost returns the X-Forwarded-Host to use for the request Host.
// By default (http.forwarded_host_trusted_only=false, as before) every peer's header
// is used. With TRUSTED_FORWARDED_HOST_ONLY=true it is used only when the TCP peer is
// in http.trusted_proxies, so a client cannot pick a tenant subdomain by forging it.
func TrustedForwardedHost(ctx http.Context) string {
	if ctx == nil {
		return ""
	}
	if facades.Config().GetBool("http.forwarded_host_trusted_only", false) &&
		!proxies.FromConfig().Contains(peerIP(ctx)) {
		return ""
	}
	return ctx.Request().Header("X-Forwarded-Host", "")
}

// peerIP 返回 TCP 对端地址（不经过 gin ClientIP，后者默认信任所有代理头）。
func peerIP(ctx http.Context) string {
	if req := ctx.Request().Origin(); req != nil {
		if parsedIP := proxies.ParseIP(req.RemoteAddr); parsedIP != "" {
			return parsedIP
		}
	}
	remoteAddr := ctx.Request().Ip()
	if parsedIP := proxies.ParseIP(remoteAddr); parsedIP != "" {
		return parsedIP
	}
	return remoteAddr
}

// clientFromForwardedFor 从右往左跳过可信代理，返回第一个不可信地址；全部可信时返回最左侧地址。
func clientFromForwardedFor(header string, trusted *proxies.Set) string {
	parts := strings.Split(header, ",")
	leftmost := ""
	for i := len(parts) - 1; i >= 0; i-- {
		ip := proxies.ParseIP(parts[i])
		if ip == "" {
			return leftmost
		}
		leftmost = ip
		if !trusted.Contains(ip) {
			return ip
		}
	}
	return leftmost
}
