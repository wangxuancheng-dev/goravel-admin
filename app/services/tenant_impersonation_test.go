package services

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"goravel/app/models"
)

func TestImpersonationTokenNameRoundTrip(t *testing.T) {
	name := ImpersonationTokenName(7, "ops:alice")
	assert.True(t, IsImpersonationTokenName(name))
	assert.False(t, IsImpersonationTokenName("admin-token"))

	exp := time.Now().Add(30 * time.Minute)
	info := ParseImpersonationToken(&models.PersonalAccessToken{Name: name, ExpiresAt: &exp})
	if assert.NotNil(t, info) {
		assert.Equal(t, uint(7), info.PlatformAdminID)
		assert.Equal(t, "ops:alice", info.PlatformUsername) // only the first ':' splits
		assert.Equal(t, &exp, info.ExpiresAt)
	}
	assert.Nil(t, ParseImpersonationToken(&models.PersonalAccessToken{Name: "admin-token"}))
	assert.Nil(t, ParseImpersonationToken(nil))

	long := ImpersonationTokenName(1, strings.Repeat("x", 400))
	assert.LessOrEqual(t, len(long), 255)
}

func TestImpersonationTarget(t *testing.T) {
	host, path := ImpersonationTarget(TenantLoginLinks{Resolver: "header", TenantCode: "acme", SubdomainURL: "https://acme.example.com/login"}, "tk")
	assert.Empty(t, host, "header mode stays on the console origin")
	assert.Equal(t, "/login/impersonate?tenant_code=acme&ticket=tk", path)

	host, _ = ImpersonationTarget(TenantLoginLinks{Resolver: "subdomain", TenantCode: "acme"}, "tk")
	assert.Empty(t, host, "subdomain mode without a base domain falls back to tenant_code")

	host, path = ImpersonationTarget(TenantLoginLinks{Resolver: "subdomain", TenantCode: "acme", SubdomainURL: "https://acme.example.com/login"}, "tk")
	assert.Equal(t, "acme.example.com", host)
	assert.Equal(t, "/login/impersonate?tenant_code=acme&ticket=tk", path)

	host, _ = ImpersonationTarget(TenantLoginLinks{Resolver: "header", TenantCode: "acme", PrimaryCustomURL: "https://crm.acme.test/login"}, "tk")
	assert.Equal(t, "crm.acme.test", host, "a custom domain resolves the tenant in header mode too")
}

func TestBuildImpersonationURL(t *testing.T) {
	// Header resolver, no custom domain: same-origin path with tenant_code.
	got := BuildImpersonationURL(TenantLoginLinks{Resolver: "header", TenantCode: "acme", SubdomainURL: "https://acme.example.com/login"}, "tk")
	assert.Equal(t, "/login/impersonate?tenant_code=acme&ticket=tk", got)

	// Subdomain resolver uses the subdomain.
	got = BuildImpersonationURL(TenantLoginLinks{Resolver: "subdomain", TenantCode: "acme", SubdomainURL: "https://acme.example.com/login"}, "tk")
	assert.Equal(t, "https://acme.example.com/login/impersonate?tenant_code=acme&ticket=tk", got)

	// A primary custom domain wins in any mode.
	got = BuildImpersonationURL(TenantLoginLinks{
		Resolver:         "subdomain",
		TenantCode:       "acme",
		SubdomainURL:     "https://acme.example.com/login",
		PrimaryCustomURL: "https://admin.acme.test/login",
	}, "tk")
	assert.Equal(t, "https://admin.acme.test/login/impersonate?tenant_code=acme&ticket=tk", got)
}
