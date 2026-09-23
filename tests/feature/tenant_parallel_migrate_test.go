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

// TestParallelTenantMigrate creates temporary tenant DBs, migrates them concurrently,
// checks isolation, then drops the DBs and platform rows.
func TestParallelTenantMigrate(t *testing.T) {
	withTenancyDriver(t, "database")
	require.True(t, tenancy.Enabled())

	// Same-host CREATE may need shared platform credentials in local/dev.
	prevAllow := facades.Config().GetBool("tenancy.allow_platform_db_credentials", false)
	facades.Config().Add("tenancy.allow_platform_db_credentials", true)
	t.Cleanup(func() {
		facades.Config().Add("tenancy.allow_platform_db_credentials", prevAllow)
	})

	// Router must be live for true parallel path.
	appfacades.InstallTenantAwareSchema(facades.App())
	// Reset any leftover Schema connection from prior tests in this package.
	if def := facades.Config().GetString("database.default", "mysql"); def != "" {
		facades.Schema().SetConnection(def)
	}

	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1_000_000_000)
	codes := []string{"pmig" + suffix + "a", "pmig" + suffix + "b", "pmig" + suffix + "c"}

	admin := services.NewTenantAdminService()
	conn := services.NewTenantConnectionService()

	tenants := make([]*models.Tenant, 0, len(codes))
	cleanup := func() {
		for _, tn := range tenants {
			if tn == nil || tn.ID == 0 {
				continue
			}
			_ = conn.DropStorage(tn)
			_, _ = appfacades.PlatformOrmQuery(nil).Where("id", tn.ID).ForceDelete(&models.Tenant{})
		}
	}
	t.Cleanup(cleanup)

	for i, code := range codes {
		tn, err := admin.Create(services.TenantCreateInput{
			Code:    code,
			Name:    fmt.Sprintf("Parallel Migrate %d", i),
			Migrate: false,
		})
		if err != nil {
			t.Skipf("skip parallel migrate: cannot create tenant DB: %v", err)
		}
		tenants = append(tenants, tn)
	}

	var wg sync.WaitGroup
	errs := make([]error, len(tenants))
	gate := make(chan struct{})

	for i := range tenants {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-gate
			errs[i] = conn.MigrateTenant(tenants[i])
		}(i)
	}
	close(gate)
	wg.Wait()

	for i, err := range errs {
		require.NoError(t, err, "migrate tenant %s", tenants[i].Code)
		require.Equal(t, models.TenantProvisionReady, tenants[i].ProvisionStatus)
		require.Greater(t, tenants[i].SchemaMigrationCount, int64(0))
	}

	// Write a marker in each tenant DB and ensure no cross-leak.
	for _, tn := range tenants {
		marker := "parallel_mig_" + tn.Code
		err := conn.WithTenantConnection(tn, func() error {
			bound := tenancyctx.WithTenant(context.Background(), tn.ID, tn.ConnectionName, tn.Code)
			return appfacades.OrmQuery(bound).Table("configs").Create(map[string]any{
				"key":    marker,
				"value":  tn.Code,
				"group":  "test",
				"type":   "string",
				"remark": "parallel migrate isolation",
			})
		})
		require.NoError(t, err)
	}

	for _, tn := range tenants {
		for _, other := range tenants {
			marker := "parallel_mig_" + other.Code
			var n int64
			err := conn.WithTenantConnection(tn, func() error {
				bound := tenancyctx.WithTenant(context.Background(), tn.ID, tn.ConnectionName, tn.Code)
				var count int64
				var qErr error
				count, qErr = appfacades.OrmQuery(bound).Table("configs").Where("key", marker).Count()
				n = count
				return qErr
			})
			require.NoError(t, err)
			if tn.ID == other.ID {
				assert.Equal(t, int64(1), n, "%s should have own marker", tn.Code)
			} else {
				assert.Equal(t, int64(0), n, "%s must not see marker from %s", tn.Code, other.Code)
			}
		}
	}
}
