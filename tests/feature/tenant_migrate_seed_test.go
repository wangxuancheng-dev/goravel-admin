package feature_test

import (
	"context"
	"fmt"
	"sync"
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

// TestTenantMigrateAndSeed creates temp tenant DBs, parallel-migrates, seeds (serial +
// concurrent seed fan-in), verifies admins land in the right DB, then drops everything.
func TestTenantMigrateAndSeed(t *testing.T) {
	withTenancyDriver(t, "database")
	require.True(t, tenancy.Enabled())

	prevAllow := facades.Config().GetBool("tenancy.allow_platform_db_credentials", false)
	facades.Config().Add("tenancy.allow_platform_db_credentials", true)
	t.Cleanup(func() {
		facades.Config().Add("tenancy.allow_platform_db_credentials", prevAllow)
	})

	appfacades.InstallTenantAwareSchema(facades.App())
	appfacades.InstallTenantAwareOrm(facades.App())
	require.True(t, appfacades.TenantAwareSchemaInstalled(), "schema router must be installed")
	require.True(t, appfacades.TenantAwareOrmInstalled(), "orm router must be installed")
	if def := facades.Config().GetString("database.default", "mysql"); def != "" {
		facades.Schema().SetConnection(def)
	}

	// Platform DB should not gain tenant seed rows on admins (if that table exists there).
	platformAdminsBefore := int64(-1)
	if facades.Schema().HasTable("admins") {
		// Ensure we are on platform default connection for this probe.
		if def := facades.Config().GetString("database.default", "mysql"); def != "" {
			facades.Schema().SetConnection(def)
		}
		n, err := appfacades.PlatformOrmQuery(nil).Table("admins").Count()
		require.NoError(t, err)
		platformAdminsBefore = n
	}

	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1_000_000_000)
	codes := []string{"msd" + suffix + "a", "msd" + suffix + "b", "msd" + suffix + "c"}

	adminSvc := services.NewTenantAdminService()
	conn := services.NewTenantConnectionService()

	tenants := make([]*models.Tenant, 0, len(codes))
	t.Cleanup(func() {
		for _, tn := range tenants {
			if tn == nil || tn.ID == 0 {
				continue
			}
			_ = conn.DropStorage(tn)
			_, _ = appfacades.PlatformOrmQuery(nil).Where("id", tn.ID).ForceDelete(&models.Tenant{})
		}
		if def := facades.Config().GetString("database.default", "mysql"); def != "" {
			facades.Schema().SetConnection(def)
		}
	})

	for i, code := range codes {
		tn, err := adminSvc.Create(services.TenantCreateInput{
			Code:    code,
			Name:    fmt.Sprintf("MigrateSeed %d", i),
			Migrate: false,
		})
		if err != nil {
			t.Skipf("skip migrate+seed: cannot create tenant DB: %v", err)
		}
		tenants = append(tenants, tn)
	}

	// --- parallel migrate ---
	migErrs := make([]error, len(tenants))
	var wg sync.WaitGroup
	gate := make(chan struct{})
	for i := range tenants {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-gate
			migErrs[i] = conn.MigrateTenant(tenants[i])
		}(i)
	}
	close(gate)
	wg.Wait()
	for i, e := range migErrs {
		require.NoError(t, e, "migrate %s", tenants[i].Code)
		require.Equal(t, models.TenantProvisionReady, tenants[i].ProvisionStatus)
	}

	// After migrate, admins table exists but should be empty before seed.
	for _, tn := range tenants {
		var n int64
		err := conn.WithTenantConnection(tn, func() error {
			bound := tenancyctx.WithTenant(context.Background(), tn.ID, tn.ConnectionName, tn.Code)
			var qErr error
			n, qErr = appfacades.OrmQuery(bound).Table("admins").Count()
			return qErr
		})
		require.NoError(t, err)
		assert.Equal(t, int64(0), n, "%s admins should be empty before seed", tn.Code)
	}

	// --- concurrent seed (true parallel when Schema+Orm routers installed) ---
	seedErrs := make([]error, len(tenants))
	gate2 := make(chan struct{})
	for i := range tenants {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-gate2
			seedErrs[i] = conn.SeedTenant(tenants[i], "AdminSeeder")
		}(i)
	}
	close(gate2)
	wg.Wait()
	for i, e := range seedErrs {
		require.NoError(t, e, "seed %s", tenants[i].Code)
	}

	// Each tenant has seeded admin; markers stay isolated.
	for _, tn := range tenants {
		var adminCount int64
		var hasAdmin bool
		err := conn.WithTenantConnection(tn, func() error {
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
		assert.Greater(t, adminCount, int64(0), "%s should have seeded admins", tn.Code)
		assert.True(t, hasAdmin, "%s should have username=admin", tn.Code)

		marker := "seed_iso_" + tn.Code
		err = conn.WithTenantConnection(tn, func() error {
			bound := tenancyctx.WithTenant(context.Background(), tn.ID, tn.ConnectionName, tn.Code)
			return appfacades.OrmQuery(bound).Table("configs").Create(map[string]any{
				"key":    marker,
				"value":  tn.Code,
				"group":  "test",
				"type":   "string",
				"remark": "migrate+seed isolation",
			})
		})
		require.NoError(t, err)
	}

	for _, tn := range tenants {
		for _, other := range tenants {
			marker := "seed_iso_" + other.Code
			var n int64
			err := conn.WithTenantConnection(tn, func() error {
				bound := tenancyctx.WithTenant(context.Background(), tn.ID, tn.ConnectionName, tn.Code)
				var qErr error
				n, qErr = appfacades.OrmQuery(bound).Table("configs").Where("key", marker).Count()
				return qErr
			})
			require.NoError(t, err)
			if tn.ID == other.ID {
				assert.Equal(t, int64(1), n)
			} else {
				assert.Equal(t, int64(0), n, "%s must not see config from %s", tn.Code, other.Code)
			}
		}
	}

	if platformAdminsBefore >= 0 {
		if def := facades.Config().GetString("database.default", "mysql"); def != "" {
			facades.Schema().SetConnection(def)
		}
		platformAdminsAfter, err := appfacades.PlatformOrmQuery(nil).Table("admins").Count()
		require.NoError(t, err)
		assert.Equal(t, platformAdminsBefore, platformAdminsAfter, "tenant seed must not write platform admins")
	}

	// Idempotent re-seed should still succeed.
	require.NoError(t, conn.SeedTenant(tenants[0], "AdminSeeder"))
}
