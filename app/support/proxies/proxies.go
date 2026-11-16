// Package proxies parses http.trusted_proxies and decides whether a peer may
// supply forwarding headers (X-Forwarded-For, X-Forwarded-Host, ...).
package proxies

import (
	"net"
	"strings"
	"sync"

	"github.com/goravel/framework/facades"
)

const Default = "127.0.0.0/8,::1/128,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16,fc00::/7"

// Set is a parsed allowlist of proxy peers.
type Set struct {
	All  bool
	Nets []*net.IPNet
}

// Contains reports whether ip (host form, no port) is a trusted proxy.
func (p *Set) Contains(ip string) bool {
	if p == nil {
		return false
	}
	if p.All {
		return true
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	for _, n := range p.Nets {
		if n.Contains(parsed) {
			return true
		}
	}
	return false
}

// Parse builds a Set from a comma-separated list of IPs / CIDRs ("*" = trust all).
func Parse(raw string) *Set {
	set := &Set{}
	for _, item := range strings.Split(raw, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if item == "*" {
			set.All = true
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
			set.Nets = append(set.Nets, n)
		}
	}
	return set
}

// ParseIP normalizes an address that may include a port to a canonical IP string.
func ParseIP(ip string) string {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return ""
	}
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return ""
	}
	if v4 := parsed.To4(); v4 != nil {
		return v4.String()
	}
	if parsed.IsLoopback() {
		return "127.0.0.1"
	}
	return parsed.String()
}

var (
	mu     sync.Mutex
	raw    string
	cached *Set
)

// FromConfig returns the process allowlist (cached until the config string changes).
func FromConfig() *Set {
	cfg := facades.Config().GetString("http.trusted_proxies", Default)
	mu.Lock()
	defer mu.Unlock()
	if cached != nil && cfg == raw {
		return cached
	}
	raw = cfg
	cached = Parse(cfg)
	return cached
}

// ResetForTest clears the cache (tests that change http.trusted_proxies).
func ResetForTest() {
	mu.Lock()
	defer mu.Unlock()
	raw = ""
	cached = nil
}
