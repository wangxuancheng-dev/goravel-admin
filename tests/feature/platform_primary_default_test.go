package feature_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/services"
)

// Making a domain primary must clear the tenant's previous primary. The clear used
// Where("id", "!=", id), which the ORM turns into id IN ('id','!=',id) and MySQL rejects,
// so a tenant ended up with two primaries.
func TestSetPrimaryDomainClearsPreviousPrimary(t *testing.T) {
	withTenancyDriver(t, "database")
	suffix := time.Now().UnixNano() % 1_000_000
	tenant := insertTenantFixture(t, fmt.Sprintf("prim%d", suffix), models.TenantProvisionReady)

	addDomain := func(host string, primary bool) *models.TenantDomain {
		d := models.TenantDomain{TenantID: tenant.ID, Host: host, IsPrimary: primary, Status: models.TenantDomainStatusActive}
		require.NoError(t, appfacades.PlatformOrmQuery(nil).Create(&d))
		return &d
	}
	first := addDomain(fmt.Sprintf("a%d.prim.test", suffix), true)
	second := addDomain(fmt.Sprintf("b%d.prim.test", suffix), false)
	t.Cleanup(func() {
		_, _ = appfacades.PlatformOrmQuery(nil).Where("tenant_id", tenant.ID).ForceDelete(&models.TenantDomain{})
	})

	_, err := services.NewTenantDomainService().SetPrimary(tenant.ID, second.ID)
	require.NoError(t, err)

	var primaries []models.TenantDomain
	require.NoError(t, appfacades.PlatformOrmQuery(nil).
		Where("tenant_id", tenant.ID).Where("is_primary", true).Get(&primaries))
	require.Len(t, primaries, 1, "exactly one primary domain")
	assert.Equal(t, second.ID, primaries[0].ID)
	assert.NotEqual(t, first.ID, primaries[0].ID)
}

// Saving a plan as default must clear is_default on every other plan (same Where bug).
func TestUpsertDefaultPlanClearsOtherDefaults(t *testing.T) {
	withTenancyDriver(t, "database")
	svc := services.NewEntitlementService(context.Background())
	require.NoError(t, svc.EnsureBootstrap())

	var prevDefaults []models.PlatformPlan
	require.NoError(t, appfacades.PlatformOrmQuery(nil).Where("is_default", true).Get(&prevDefaults))
	code := fmt.Sprintf("dflt%d", time.Now().UnixNano()%1_000_000)
	t.Cleanup(func() {
		_, _ = appfacades.PlatformOrmQuery(nil).Where("code", code).ForceDelete(&models.PlatformPlan{})
		for _, p := range prevDefaults {
			_, _ = appfacades.PlatformOrmQuery(nil).Model(&models.PlatformPlan{}).
				Where("id", p.ID).Update("is_default", true)
		}
	})

	plan, err := svc.UpsertPlan(services.UpsertPlanInput{Code: code, Name: "Default test", IsDefault: true})
	require.NoError(t, err)

	var defaults []models.PlatformPlan
	require.NoError(t, appfacades.PlatformOrmQuery(nil).Where("is_default", true).Get(&defaults))
	require.Len(t, defaults, 1, "exactly one default plan")
	assert.Equal(t, plan.ID, defaults[0].ID)
}
