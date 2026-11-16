package helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"goravel/app/support/proxies"
)

func TestClientFromForwardedForSkipsTrustedFromRight(t *testing.T) {
	trusted := proxies.Parse(proxies.Default)

	// nginx appends the real peer; the left part is whatever the client sent.
	assert.Equal(t, "203.0.113.7", clientFromForwardedFor("1.2.3.4, 203.0.113.7", trusted))
	// Internal hops on the right are skipped.
	assert.Equal(t, "203.0.113.7", clientFromForwardedFor("1.2.3.4, 203.0.113.7, 10.0.0.2, 172.18.0.5", trusted))
	// Garbage cannot be used to reach further left.
	assert.Equal(t, "10.0.0.2", clientFromForwardedFor("1.2.3.4, not-an-ip, 10.0.0.2", trusted))
	// All hops trusted: fall back to the leftmost.
	assert.Equal(t, "192.168.1.10", clientFromForwardedFor("192.168.1.10, 10.0.0.2", trusted))
}

func TestParseProxySet(t *testing.T) {
	set := proxies.Parse("203.0.113.0/24, 198.51.100.9, ::1, bogus")
	assert.True(t, set.Contains("203.0.113.200"))
	assert.True(t, set.Contains("198.51.100.9"))
	assert.False(t, set.Contains("198.51.100.10"))
	assert.True(t, set.Contains("::1"))
	assert.False(t, set.Contains("8.8.8.8"))

	assert.True(t, proxies.Parse("*").Contains("8.8.8.8"))
	assert.False(t, proxies.Parse("").Contains("127.0.0.1"))
}
