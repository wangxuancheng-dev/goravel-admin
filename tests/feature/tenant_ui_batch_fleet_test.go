package feature_test

import (
	"context"
	"fmt"
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

// TestUIBatchMigrateFleet creates several tenant DBs then runs the same path as
// platform MigrateBatch: BeginQueuedOp per tenant + RunTenantOpsFleet (inline,
// no Redis) with TENANCY_MIGRATE_CONCURRENCY parallelism and with_seed.
func TestUIBatchMigrateFleet(t *testing.T) {
	withTenancyDriver(t, "database")
	require.True(t, tenancy.Enabled())

	prevAllow := facades.Config().GetBool("tenancy.allow_platform_db_credentials", false)
	facades.Config().Add("tenancy.allow_platform_db_credentials", true)
	t.Cleanup(func() {
		facades.Config().Add("tenancy.allow_platform_db_credentials", prevAllow)
	})

	prevConc := facades.Config().GetInt("tenancy.migrate_concurrency", 2)
	facades.Config().Add("tenancy.migrate_concurrency", 4)
	t.Cleanup(func() {
		facades.Config().Add("tenancy.migrate_concurrency", prevConc)
	})

	appfacades.InstallTenantAwareSchema(facades.App())
	appfacades.InstallTenantAwareOrm(facades.App())
	if def := facades.Config().GetString("database.default", "mysql"); def != "" {
		facades.Schema().SetConnection(def)
	}

	// Inline fleet (tests without queue worker).
	prevFleet := services.EnqueueTenantOpsFleetFn
	prevOps := services.EnqueueTenantOpsFn
	services.EnqueueTenantOpsFleetFn = nil
	services.EnqueueTenantOpsFn = nil
	t.Cleanup(func() {
		services.EnqueueTenantOpsFleetFn = prevFleet
		services.EnqueueTenantOpsFn = prevOps
	})

	n := 4
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1_000_000_000)
	adminSvc := services.NewTenantAdminService()
	conn := services.NewTenantConnectionService()
	ops := services.NewTenantOpsService()

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
		code := fmt.Sprintf("uib%s%02d", suffix, i)
		tn, err := adminSvc.Create(services.TenantCreateInput{
			Code:    code,
			Name:    fmt.Sprintf("UI Batch %d", i),
			Migrate: false,
		})
		if err != nil {
			t.Skipf("skip UI batch fleet: cannot create tenant DB: %v", err)
		}
		tenants = append(tenants, tn)
	}

	batchID := services.NewTenantOpsBatchID()
	actor := services.TenantOpActor{ID: 1, Name: "test"}
	items := make([]services.TenantOpsArgs, 0, n)
	for _, tn := range tenants {
		_, args, err := ops.BeginQueuedOp(tn.ID, models.TenantOpMigrate, true, actor, batchID)
		require.NoError(t, err, "BeginQueuedOp %s", tn.Code)
		items = append(items, args)
	}

	start := time.Now()
	require.NoError(t, services.EnqueuePreparedTenantOpsBatch(items))
	dur := time.Since(start)
	t.Logf("UI batch fleet: %d tenants concurrency=%d wall=%s batch=%s", n, services.FleetConcurrencyForResponse(), dur.Round(time.Millisecond), batchID)

	for _, tn := range tenants {
		fresh, err := adminSvc.GetByID(tn.ID)
		require.NoError(t, err)
		require.Equal(t, models.TenantProvisionReady, fresh.ProvisionStatus, "%s provision", tn.Code)
		require.Equal(t, models.TenantOpStatusSuccess, fresh.LastOpStatus, "%s last_op_status=%s msg=%s", tn.Code, fresh.LastOpStatus, fresh.LastOpMessage)
		require.Greater(t, fresh.SchemaMigrationCount, int64(0))

		var adminCount int64
		err = conn.WithTenantConnection(fresh, func() error {
			bound := tenancyctx.WithTenant(context.Background(), fresh.ID, fresh.ConnectionName, fresh.Code)
			var qErr error
			adminCount, qErr = appfacades.OrmQuery(bound).Table("admins").Count()
			return qErr
		})
		require.NoError(t, err)
		assert.Greater(t, adminCount, int64(0), "%s should be seeded", tn.Code)
	}
}