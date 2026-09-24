package feature_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/goravel/framework/facades"
	"github.com/stretchr/testify/require"

	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/app/tenancy"
)

// TestBackupFleetFanOut creates several ready tenant DBs, runs FanOutTenantBackups
// (same path as tenant:backup-all / scheduled full) with inline fleet +
// TENANCY_BACKUP_CONCURRENCY, and checks dump files + last_op success.
func TestBackupFleetFanOut(t *testing.T) {
	withTenancyDriver(t, "database")
	require.True(t, tenancy.Enabled())

	prevAllow := facades.Config().GetBool("tenancy.allow_platform_db_credentials", false)
	facades.Config().Add("tenancy.allow_platform_db_credentials", true)
	t.Cleanup(func() {
		facades.Config().Add("tenancy.allow_platform_db_credentials", prevAllow)
	})

	prevConc := facades.Config().GetInt("tenancy.backup_concurrency", 2)
	facades.Config().Add("tenancy.backup_concurrency", 3)
	t.Cleanup(func() {
		facades.Config().Add("tenancy.backup_concurrency", prevConc)
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

	tenants := make([]*models.Tenant, 0, n)
	t.Cleanup(func() {
		for _, tn := range tenants {
			if tn == nil || tn.ID == 0 {
				continue
			}
			_ = os.RemoveAll(services.TenantBackupDir(tn.Code))
			_ = conn.DropStorage(tn)
			_, _ = appfacades.PlatformOrmQuery(nil).Where("id", tn.ID).ForceDelete(&models.Tenant{})
		}
		if def := facades.Config().GetString("database.default", "mysql"); def != "" {
			facades.Schema().SetConnection(def)
		}
	})

	for i := 0; i < n; i++ {
		code := fmt.Sprintf("bkf%s%02d", suffix, i)
		tn, err := adminSvc.Create(services.TenantCreateInput{
			Code:    code,
			Name:    fmt.Sprintf("Backup Fleet %d", i),
			Migrate: true,
		})
		if err != nil {
			t.Skipf("skip backup fleet: cannot create/migrate tenant DB: %v", err)
		}
		tenants = append(tenants, tn)
		require.Equal(t, models.TenantProvisionReady, tn.ProvisionStatus, code)
	}

	start := time.Now()
	batchID := services.NewTenantOpsBatchID()
	ops := services.NewTenantOpsService()
	items := make([]services.TenantOpsArgs, 0, n)
	for _, tn := range tenants {
		_, args, beginErr := ops.BeginQueuedBackup(tn.ID, 3, services.TenantOpActor{Name: "test:backup-fleet"}, batchID)
		require.NoError(t, beginErr, tn.Code)
		items = append(items, args)
	}
	require.NoError(t, services.EnqueuePreparedTenantOpsBatch(items))
	dur := time.Since(start)
	t.Logf("backup fleet: %d tenants concurrency=%d wall=%s batch=%s", n, tenancy.BackupConcurrency(), dur.Round(time.Millisecond), batchID)

	for _, tn := range tenants {
		fresh, err := adminSvc.GetByID(tn.ID)
		require.NoError(t, err)
		require.Equal(t, models.TenantOpBackup, fresh.LastOp, tn.Code)
		require.Equal(t, models.TenantOpStatusSuccess, fresh.LastOpStatus, "%s msg=%s", tn.Code, fresh.LastOpMessage)
		require.NotEmpty(t, fresh.LastBackupPath, tn.Code)

		dir := services.TenantBackupDir(tn.Code)
		entries, readErr := os.ReadDir(dir)
		require.NoError(t, readErr, dir)
		require.NotEmpty(t, entries, "expected dump under %s", dir)
		// At least one .sql file with non-trivial size
		var found bool
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			info, _ := e.Info()
			if filepath.Ext(e.Name()) == ".sql" && info != nil && info.Size() > 100 {
				found = true
				t.Logf("%s dump %s (%d bytes)", tn.Code, e.Name(), info.Size())
				break
			}
		}
		require.True(t, found, "no .sql dump in %s", dir)
	}
}
