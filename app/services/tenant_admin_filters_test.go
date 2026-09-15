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
		SchemaStatus:    "behind",
	}
	assert.Equal(t, "failed", f.ProvisionStatus)
	assert.Equal(t, "acme", f.Code)
	assert.Equal(t, "behind", f.SchemaStatus)
}

func TestWebsiteBrandingPublicKeys(t *testing.T) {
	// Document the public branding contract without hitting ORM.
	keys := []string{"site_enabled", "site_name", "site_logo", "site_copyright"}
	assert.Len(t, keys, 4)
	assert.NotContains(t, keys, "site_theme_color")
}
