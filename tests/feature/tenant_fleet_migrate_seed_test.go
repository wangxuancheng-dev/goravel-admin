package feature_test

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
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

// TestFleetTenantMigrateAndSeed creates many temp tenant DBs, runs parallel
// migrate then parallel seed (same semaphore pattern as tenant:migrate-all /
// seed-all), checks isolation, then drops everything.
//
// Defaults: 10 tenants, concurrency 5. Override with:
//
//	TENANCY_TEST_FLEET_COUNT=20 TENANCY_TEST_FLEET_CONCURRENCY=8 go test ...
func TestFleetTenantMigrateAndSeed(t *testing.T) {
	withTenancyDriver(t, "database")
	require.True(t, tenancy.Enabled())

	prevAllow := facades.Config().GetBool("tenancy.allow_platform_db_credentials", false)
	facades.Config().Add("tenancy.allow_platform_db_credentials", true)
	t.Cleanup(func() {
		facades.Config().Add("tenancy.allow_platform_db_credentials", prevAllow)
	})

	appfacades.InstallTenantAwareSchema(facades.App())
	appfacades.InstallTenantAwareOrm(facades.App())
	require.True(t, appfacades.TenantAwareSchemaInstalled())
	require.True(t, appfacades.TenantAwareOrmInstalled())
	if def := facades.Config().GetString("database.default", "mysql"); def != "" {
		facades.Schema().SetConnection(def)
	}

	n := fleetTestIntEnv("TENANCY_TEST_FLEET_COUNT", 10)
	concurrency := tenancy.ClampMigrateConcurrency(fleetTestIntEnv("TENANCY_TEST_FLEET_CONCURRENCY", 5))
	if n < 2 {
		n = 2
	}
	if n > 50 {
		// Cap to keep local MySQL from melting; raise only via env after edit.
		n = 50
	}
	t.Logf("fleet: tenants=%d concurrency=%d", n, concurrency)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1_000_000_000)
	adminSvc := services.NewTenantAdminService()
	conn := services.NewTenantConnectionService()

	tenants := make([]*models.Tenant, 0, n)
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

	for i := 0; i < n; i++ {
		code := fmt.Sprintf("flt%s%02d", suffix, i)
		tn, err := adminSvc.Create(services.TenantCreateInput{
			Code:    code,
			Name:    fmt.Sprintf("Fleet %d", i),
			Migrate: false,
		})
		if err != nil {
			t.Skipf("skip fleet migrate+seed: cannot create tenant DB (%d/%d): %v", i+1, n, err)
		}
		tenants = append(tenants, tn)
	}
	require.Len(t, tenants, n)

	// --- parallel migrate (migrate-all style) ---
	migStart := time.Now()
	migErrs := runFleetParallel(tenants, concurrency, func(tn *models.Tenant) error {
		return conn.MigrateTenant(tn)
	})
	migDur := time.Since(migStart)
	for i, e := range migErrs {
		require.NoError(t, e, "migrate %s", tenants[i].Code)
		require.Equal(t, models.TenantProvisionReady, tenants[i].ProvisionStatus)
		require.Greater(t, tenants[i].SchemaMigrationCount, int64(0))
	}
	t.Logf("parallel migrate: %d tenants in %s (concurrency=%d)", n, migDur.Round(time.Millisecond), concurrency)

	// Empty before seed.
	for _, tn := range tenants {
		var count int64
		err := conn.WithTenantConnection(tn, func() error {
			bound := tenancyctx.WithTenant(context.Background(), tn.ID, tn.ConnectionName, tn.Code)
			var qErr error
			count, qErr = appfacades.OrmQuery(bound).Table("admins").Count()
			return qErr
		})
		require.NoError(t, err)
		assert.Equal(t, int64(0), count, "%s admins empty before seed", tn.Code)
	}

	// --- parallel seed (seed-all style) ---
	seedStart := time.Now()
	seedErrs := runFleetParallel(tenants, concurrency, func(tn *models.Tenant) error {
		return conn.SeedTenant(tn, "AdminSeeder")
	})
	seedDur := time.Since(seedStart)
	for i, e := range seedErrs {
		require.NoError(t, e, "seed %s", tenants[i].Code)
	}
	t.Logf("parallel seed: %d tenants in %s (concurrency=%d)", n, seedDur.Round(time.Millisecond), concurrency)

	// Per-tenant admin + isolation markers.
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
		assert.Greater(t, adminCount, int64(0), "%s seeded admins", tn.Code)
		assert.True(t, hasAdmin, "%s has username=admin", tn.Code)

		marker := "fleet_iso_" + tn.Code
		err = conn.WithTenantConnection(tn, func() error {
			bound := tenancyctx.WithTenant(context.Background(), tn.ID, tn.ConnectionName, tn.Code)
			return appfacades.OrmQuery(bound).Table("configs").Create(map[string]any{
				"key":    marker,
				"value":  tn.Code,
				"group":  "test",
				"type":   "string",
				"remark": "fleet migrate+seed isolation",
			})
		})
		require.NoError(t, err)
	}

	// Spot-check isolation: first vs last vs middle (full NxN is O(n^2) queries).
	checkPairs := [][2]int{{0, 0}, {0, n - 1}, {n / 2, 0}, {n - 1, n - 1}}
	if n > 3 {
		checkPairs = append(checkPairs, [2]int{1, 2}, [2]int{n - 2, 1})
	}
	for _, pair := range checkPairs {
		tn := tenants[pair[0]]
		other := tenants[pair[1]]
		marker := "fleet_iso_" + other.Code
		var count int64
		err := conn.WithTenantConnection(tn, func() error {
			bound := tenancyctx.WithTenant(context.Background(), tn.ID, tn.ConnectionName, tn.Code)
			var qErr error
			count, qErr = appfacades.OrmQuery(bound).Table("configs").Where("key", marker).Count()
			return qErr
		})
		require.NoError(t, err)
		if tn.ID == other.ID {
			assert.Equal(t, int64(1), count)
		} else {
			assert.Equal(t, int64(0), count, "%s must not see %s", tn.Code, other.Code)
		}
	}
}

func runFleetParallel(tenants []*models.Tenant, concurrency int, fn func(*models.Tenant) error) []error {
	errs := make([]error, len(tenants))
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var started atomic.Int64
	gate := make(chan struct{})

	for i := range tenants {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-gate
			sem <- struct{}{}
			defer func() { <-sem }()
			started.Add(1)
			errs[i] = fn(tenants[i])
		}(i)
	}
	close(gate)
	wg.Wait()
	_ = started.Load()
	return errs
}

func fleetTestIntEnv(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return fallback
	}
	return n
}