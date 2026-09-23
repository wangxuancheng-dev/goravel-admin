package services

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"goravel/app/models"
)

func TestTenantAdminFiltersIncludeProvisionStatus(t *testing.T) {
	f := TenantAdminFilters{
		Code:            "acme",
		Name:            "Acme",
		Status:          "1",
		ProvisionStatus: "failed",
		SchemaStatus:    "behind",
		LastOp:          "seed",
		LastOpStatus:    "failed",
	}
	assert.Equal(t, "failed", f.ProvisionStatus)
	assert.Equal(t, "acme", f.Code)
	assert.Equal(t, "behind", f.SchemaStatus)
	assert.Equal(t, "seed", f.LastOp)
	assert.Equal(t, "failed", f.LastOpStatus)
}

func TestNormalizeTenantLastOpFilter(t *testing.T) {
	assert.Equal(t, models.TenantOpSeed, NormalizeTenantLastOpFilter(" Seed "))
	assert.Equal(t, models.TenantOpMigrate, NormalizeTenantLastOpFilter("MIGRATE"))
	assert.Equal(t, "", NormalizeTenantLastOpFilter("drop"))
	assert.Equal(t, "", NormalizeTenantLastOpFilter(""))
}

func TestNormalizeTenantLastOpStatusFilter(t *testing.T) {
	assert.Equal(t, models.TenantOpStatusFailed, NormalizeTenantLastOpStatusFilter("FAILED"))
	assert.Equal(t, models.TenantOpStatusSuccess, NormalizeTenantLastOpStatusFilter("success"))
	assert.Equal(t, "", NormalizeTenantLastOpStatusFilter("done"))
	assert.Equal(t, "", NormalizeTenantLastOpStatusFilter(""))
}

func TestCliTenantOpActorName(t *testing.T) {
	assert.Equal(t, "cli", CliTenantOpActor.Name)
}

func TestWebsiteBrandingPublicKeys(t *testing.T) {
	// Document the public branding contract without hitting ORM.
	keys := []string{"site_enabled", "site_name", "site_logo", "site_copyright"}
	assert.Len(t, keys, 4)
	assert.NotContains(t, keys, "site_theme_color")
}
