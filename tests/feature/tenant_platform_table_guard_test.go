package feature_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/app/tenantdb"
)

// Platform models read through a tenant ctx fail at once, naming PlatformQuery, instead
// of "table doesn't exist" (or a stale copy); tenant tables and the platform query work.
func TestTenantConnectionRefusesPlatformTables(t *testing.T) {
	withTenancyDriver(t, "database")
	allowSharedTenantDBCredentials(t)

	code := fmt.Sprintf("guard%d", time.Now().UnixNano()%1000000)
	admin := services.NewTenantAdminService()
	conn := services.NewTenantConnectionService()
	tn, err := admin.Create(services.TenantCreateInput{Code: code, Name: "Guard", Migrate: false})
	if err != nil {
		skipWithoutTenantDB(t, err)
	}
	t.Cleanup(func() {
		_ = conn.DropStorage(tn)
		_, _ = appfacades.PlatformQuery().Where("id", tn.ID).ForceDelete(&models.Tenant{})
	})
	require.NoError(t, conn.MigrateTenant(tn))

	ctx, err := conn.TenantContext(tn)
	require.NoError(t, err)

	// Model, Table, Find, Create, Update and Delete are all refused.
	_, err = appfacades.OrmQuery(ctx).Model(&models.Tenant{}).Count()
	assert.ErrorIs(t, err, tenantdb.ErrPlatformTable)
	_, err = appfacades.OrmQuery(ctx).Table("platform_plans").Count()
	assert.ErrorIs(t, err, tenantdb.ErrPlatformTable)
	var plans []models.PlatformPlan
	assert.ErrorIs(t, appfacades.OrmQuery(ctx).Find(&plans), tenantdb.ErrPlatformTable)
	assert.ErrorIs(t, appfacades.OrmQuery(ctx).Create(&models.PlatformSetting{Key: "x", Value: "y"}), tenantdb.ErrPlatformTable)
	_, err = appfacades.OrmQuery(ctx).Model(&models.TenantDomain{}).Where("id", 1).Update("host", "x")
	assert.ErrorIs(t, err, tenantdb.ErrPlatformTable)
	_, err = appfacades.OrmQuery(ctx).Where("id", 1).Delete(&models.PlatformAdmin{})
	assert.ErrorIs(t, err, tenantdb.ErrPlatformTable)

	// Tenant tables still work on the tenant connection; platform tables on the platform one.
	_, err = appfacades.OrmQuery(ctx).Model(&models.Admin{}).Count()
	require.NoError(t, err)
	_, err = appfacades.PlatformQuery().Model(&models.Tenant{}).Count()
	require.NoError(t, err)
}
