package helpers

import (
	"net"
	"strings"
	"sync"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"
	"github.com/goravel/framework/support/str"
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
	trusted := trustedProxies()
	if !trusted.contains(peer) {
		return peer
	}

	if header := strings.TrimSpace(facades.Config().GetString("http.trusted_ip_header", "")); header != "" {
		if parsedIP := parseIP(ctx.Request().Header(header, "")); parsedIP != "" {
			return parsedIP
		}
	}

	if forwardedFor := ctx.Request().Header("X-Forwarded-For", ""); !str.Of(forwardedFor).IsEmpty() {
		if ip := clientFromForwardedFor(forwardedFor, trusted); ip != "" {
			return ip
		}
	}

	if parsedIP := parseIP(ctx.Request().Header("X-Real-IP", "")); parsedIP != "" {
		return parsedIP
	}

	return peer
}

// peerIP 返回 TCP 对端地址（不经过 gin ClientIP，后者默认信任所有代理头）。
func peerIP(ctx http.Context) string {
	if req := ctx.Request().Origin(); req != nil {
		if parsedIP := parseIP(req.RemoteAddr); parsedIP != "" {
			return parsedIP
		}
	}
	remoteAddr := ctx.Request().Ip()
	if parsedIP := parseIP(remoteAddr); parsedIP != "" {
		return parsedIP
	}
	return remoteAddr
}

// clientFromForwardedFor 从右往左跳过可信代理，返回第一个不可信地址；全部可信时返回最左侧地址。
func clientFromForwardedFor(header string, trusted *proxySet) string {
	parts := strings.Split(header, ",")
	leftmost := ""
	for i := len(parts) - 1; i >= 0; i-- {
		ip := parseIP(parts[i])
		if ip == "" {
			// 无法解析的条目之前的内容都不可信
			return leftmost
		}
		leftmost = ip
		if !trusted.contains(ip) {
			return ip
		}
	}
	return leftmost
}

type proxySet struct {
	all  bool
	nets []*net.IPNet
}

func (p *proxySet) contains(ip string) bool {
	if p.all {
		return true
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	for _, n := range p.nets {
		if n.Contains(parsed) {
			return true
		}
	}
	return false
}

var (
	trustedProxiesMu  sync.Mutex
	trustedProxiesRaw string
	trustedProxiesSet *proxySet
)

// trustedProxies 解析 http.trusted_proxies（逗号分隔的 IP / CIDR，"*" 表示信任所有，不推荐）。
func trustedProxies() *proxySet {
	raw := facades.Config().GetString("http.trusted_proxies", defaultTrustedProxies)
	trustedProxiesMu.Lock()
	defer trustedProxiesMu.Unlock()
	if trustedProxiesSet != nil && raw == trustedProxiesRaw {
		return trustedProxiesSet
	}
	trustedProxiesRaw = raw
	trustedProxiesSet = parseProxySet(raw)
	return trustedProxiesSet
}

const defaultTrustedProxies = "127.0.0.0/8,::1/128,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16,fc00::/7"

func parseProxySet(raw string) *proxySet {
	set := &proxySet{}
	for _, item := range strings.Split(raw, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if item == "*" {
			set.all = true
			continue
		}
		if !strings.Contains(item, "/") {
			if ip := net.ParseIP(item); ip != nil {
				if ip.To4() != nil {
					item += "/32"
				} else {
					item += "/128"
				}
			}
		}
		if _, n, err := net.ParseCIDR(item); err == nil {
			set.nets = append(set.nets, n)
		}
	}
	return set
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
