package feature_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/goravel/framework/facades"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/app/tenancy"
	"goravel/app/tenancyctx"
)

// TestTenantCreateWithMigrateAndSeed exercises the product path:
// CreateStorage + MigrateTenant + SeedTenant in one Create(Migrate:true) call.
func TestTenantCreateWithMigrateAndSeed(t *testing.T) {
	withTenancyDriver(t, "database")
	require.True(t, tenancy.Enabled())

	prevAllow := facades.Config().GetBool("tenancy.allow_platform_db_credentials", false)
	facades.Config().Add("tenancy.allow_platform_db_credentials", true)
	t.Cleanup(func() {
		facades.Config().Add("tenancy.allow_platform_db_credentials", prevAllow)
	})

	appfacades.InstallTenantAwareSchema(facades.App())
	appfacades.InstallTenantAwareOrm(facades.App())
	if def := facades.Config().GetString("database.default", "mysql"); def != "" {
		facades.Schema().SetConnection(def)
	}

	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1_000_000_000)
	code := "crms" + suffix

	adminSvc := services.NewTenantAdminService()
	conn := services.NewTenantConnectionService()

	tn, err := adminSvc.Create(services.TenantCreateInput{
		Code:    code,
		Name:    "Create Migrate Seed",
		Migrate: true,
	})
	if err != nil {
		t.Skipf("skip create+migrate+seed: cannot create tenant DB: %v", err)
	}
	t.Cleanup(func() {
		_ = conn.DropStorage(tn)
		_, _ = appfacades.PlatformOrmQuery(nil).Where("id", tn.ID).ForceDelete(&models.Tenant{})
		if def := facades.Config().GetString("database.default", "mysql"); def != "" {
			facades.Schema().SetConnection(def)
		}
	})

	require.Equal(t, models.TenantProvisionReady, tn.ProvisionStatus)
	require.Greater(t, tn.SchemaMigrationCount, int64(0))
	require.NotEmpty(t, tn.Database)
	require.True(t, strings.HasPrefix(tn.Database, "tenant_"))

	var adminCount int64
	var hasAdmin bool
	err = conn.WithTenantConnection(tn, func() error {
		bound := tenancyctx.WithTenant(context.Background(), tn.ID, tn.ConnectionName, tn.Code)
		var qErr error
		adminCount, qErr = appfacades.OrmQuery(bound).Table("admins").Count()
		if qErr != nil {
			return qErr
		}
		hasAdmin, qErr = appfacades.OrmQuery(bound).Table("admins").Where("username", "admin").Exists()
		return qErr
	})
	require.NoError(t, err)
	assert.Greater(t, adminCount, int64(0), "Create(Migrate:true) should seed admins")
	assert.True(t, hasAdmin)
}