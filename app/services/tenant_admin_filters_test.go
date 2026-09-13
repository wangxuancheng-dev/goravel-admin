package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTenantAdminFiltersIncludeProvisionStatus(t *testing.T) {
	f := TenantAdminFilters{
		Code:            "acme",
		Name:            "Acme",
		Status:          "1",
		ProvisionStatus: "failed",
	}
	assert.Equal(t, "failed", f.ProvisionStatus)
	assert.Equal(t, "acme", f.Code)
}

func TestWebsiteBrandingKeys(t *testing.T) {
	// Ensure branding helper returns the expected public keys without panic on empty DB.
	svc := NewConfigService(nil)
	branding := svc.GetWebsiteBranding()
	assert.Contains(t, branding, "site_name")
	assert.Contains(t, branding, "site_logo")
	assert.Contains(t, branding, "site_theme_color")
	assert.Contains(t, branding, "site_copyright")
	assert.Contains(t, branding, "site_enabled")
}
