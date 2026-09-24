package feature_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/goravel/framework/facades"
	"github.com/stretchr/testify/require"

	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/app/tenancy"
)

// TestUIBatchRollbackFleet migrate then rollback step=1 via the same fleet path as
// platform OpsBatch(op=rollback). Prefer step=1: first migrate often uses batch 1 for all files.
func TestUIBatchRollbackFleet(t *testing.T) {
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

	prevFleet := services.EnqueueTenantOpsFleetFn
	prevOps := services.EnqueueTenantOpsFn
	services.EnqueueTenantOpsFleetFn = nil
	services.EnqueueTenantOpsFn = nil
	t.Cleanup(func() {
		services.EnqueueTenantOpsFleetFn = prevFleet
		services.EnqueueTenantOpsFn = prevOps
	})

	n := 3
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
		code := fmt.Sprintf("urb%s%02d", suffix, i)
		tn, err := adminSvc.Create(services.TenantCreateInput{
			Code:    code,
			Name:    fmt.Sprintf("UI Rollback %d", i),
			Migrate: false,
		})
		if err != nil {
			t.Skipf("skip UI rollback fleet: cannot create tenant DB: %v", err)
		}
		tenants = append(tenants, tn)
	}

	actor := services.TenantOpActor{ID: 1, Name: "test"}
	migrateBatch := services.NewTenantOpsBatchID()
	migrateItems := make([]services.TenantOpsArgs, 0, n)
	for _, tn := range tenants {
		_, args, err := ops.BeginQueuedOp(tn.ID, models.TenantOpMigrate, false, actor, migrateBatch)
		require.NoError(t, err, "BeginQueuedOp migrate %s", tn.Code)
		migrateItems = append(migrateItems, args)
	}
	require.NoError(t, services.EnqueuePreparedTenantOpsBatch(migrateItems))

	beforeCounts := make(map[uint]int64, n)
	for _, tn := range tenants {
		fresh, err := adminSvc.GetByID(tn.ID)
		require.NoError(t, err)
		require.Equal(t, models.TenantProvisionReady, fresh.ProvisionStatus, "%s provision", tn.Code)
		require.Greater(t, fresh.SchemaMigrationCount, int64(0), "%s should have migrations", tn.Code)
		beforeCounts[tn.ID] = fresh.SchemaMigrationCount
	}

	rollbackBatch := services.NewTenantOpsBatchID()
	rollbackItems := make([]services.TenantOpsArgs, 0, n)
	for _, tn := range tenants {
		_, args, err := ops.BeginQueuedRollback(tn.ID, 1, 0, actor, rollbackBatch)
		require.NoError(t, err, "BeginQueuedRollback %s", tn.Code)
		rollbackItems = append(rollbackItems, args)
	}

	start := time.Now()
	require.NoError(t, services.EnqueuePreparedTenantOpsBatch(rollbackItems))
	dur := time.Since(start)
	t.Logf("UI rollback fleet: %d tenants concurrency=%d wall=%s batch=%s", n, services.FleetConcurrencyForResponse(), dur.Round(time.Millisecond), rollbackBatch)

	for _, tn := range tenants {
		fresh, err := adminSvc.GetByID(tn.ID)
		require.NoError(t, err)
		require.Equal(t, models.TenantOpRollback, fresh.LastOp, tn.Code)
		require.Equal(t, models.TenantOpStatusSuccess, fresh.LastOpStatus, "%s last_op_status=%s msg=%s", tn.Code, fresh.LastOpStatus, fresh.LastOpMessage)
		require.Less(t, fresh.SchemaMigrationCount, beforeCounts[tn.ID], "%s migration count should decrease after step=1", tn.Code)
	}
}
