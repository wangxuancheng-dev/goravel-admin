package feature_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/goravel/framework/contracts/database/orm"
	"github.com/goravel/framework/facades"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/app/tenancy"
	"goravel/app/tenancyctx"
)

func configMarkerRow(key, value string) map[string]any {
	return map[string]any{
		"key":    key,
		"value":  value,
		"group":  "test",
		"type":   "string",
		"label":  "tx_test",
		"remark": "orm transaction rollback test",
	}
}

func countConfigKeyOn(q orm.Query, key string) (int64, error) {
	return q.Table("configs").Where("key", key).Count()
}

func countConfigKey(ctx context.Context, key string) (int64, error) {
	return appfacades.OrmQuery(ctx).Table("configs").Where("key", key).Count()
}

// TestOrmTransactionRollbackDefault covers platform/default connection rollback.
// Uses PlatformConnectionName explicitly so it stays valid even when TENANCY_DRIVER=database
// (does not toggle tenancy.driver mid-suite).
func TestOrmTransactionRollbackDefault(t *testing.T) {
	platform := appfacades.BuildOrmConnection(appfacades.PlatformConnectionName())
	require.NotNil(t, platform)

	ctx := context.Background()
	pq := platform.WithContext(ctx).Query()
	if !appfacades.SchemaHasTable(ctx, "configs") {
		// SchemaHasTable follows tenant-aware schema; fall back to platform query probe.
		if _, err := pq.Table("configs").Limit(1).Count(); err != nil {
			t.Skip("configs table missing on platform connection")
		}
	}

	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1_000_000_000)
	key := "tx_rb_def_" + suffix

	t.Cleanup(func() {
		_, _ = pq.Table("configs").Where("key", key).Delete(&models.Config{})
	})

	err := platform.WithContext(ctx).Transaction(func(tx orm.Query) error {
		if err := tx.Table("configs").Create(configMarkerRow(key, "should-roll-back")); err != nil {
			return err
		}
		return errors.New("force rollback")
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "force rollback")

	n, err := countConfigKeyOn(pq, key)
	require.NoError(t, err)
	assert.Equal(t, int64(0), n, "failed transaction must not leave a configs row")

	err = platform.WithContext(ctx).Transaction(func(tx orm.Query) error {
		return tx.Table("configs").Create(configMarkerRow(key, "committed"))
	})
	require.NoError(t, err)

	n, err = countConfigKeyOn(pq, key)
	require.NoError(t, err)
	assert.Equal(t, int64(1), n, "successful transaction must commit")
}

// TestOrmTransactionRollbackTenant covers multi-tenant: OrmTransaction must open
// on the tenant connection; rollback must not leave data in that tenant DB.
// Mirrors TestDualTenantDatabaseIsolation (no InstallTenantAware* / Orm.Fresh mid-suite).
func TestOrmTransactionRollbackTenant(t *testing.T) {
	withTenancyDriver(t, "database")
	require.True(t, tenancy.Enabled())

	prevAllow := facades.Config().GetBool("tenancy.allow_platform_db_credentials", false)
	facades.Config().Add("tenancy.allow_platform_db_credentials", true)
	t.Cleanup(func() {
		facades.Config().Add("tenancy.allow_platform_db_credentials", prevAllow)
	})

	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1_000_000_000)
	code := "txrb" + suffix
	key := "tx_rb_ten_" + suffix

	adminSvc := services.NewTenantAdminService()
	conn := services.NewTenantConnectionService()

	tn, err := adminSvc.Create(services.TenantCreateInput{
		Code:    code,
		Name:    "TX Rollback",
		Migrate: false,
	})
	if err != nil {
		t.Skipf("skip tenant OrmTransaction rollback: cannot create tenant DB: %v", err)
	}
	t.Cleanup(func() {
		_ = conn.DropStorage(tn)
		_, _ = appfacades.PlatformOrmQuery(nil).Where("id", tn.ID).ForceDelete(&models.Tenant{})
		if def := facades.Config().GetString("database.default", "mysql"); def != "" {
			facades.Schema().SetConnection(def)
		}
	})

	require.NoError(t, conn.MigrateTenant(tn))

	bound := tenancyctx.WithTenant(context.Background(), tn.ID, tn.ConnectionName, tn.Code)

	err = appfacades.OrmTransaction(bound, func(tx orm.Query) error {
		if err := tx.Table("configs").Create(configMarkerRow(key, "should-roll-back")); err != nil {
			return err
		}
		return errors.New("force rollback")
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "force rollback")

	var afterRollback int64
	err = conn.WithTenantConnection(tn, func() error {
		var qErr error
		afterRollback, qErr = countConfigKey(bound, key)
		return qErr
	})
	require.NoError(t, err)
	assert.Equal(t, int64(0), afterRollback, "tenant failed transaction must not leave a configs row")

	err = appfacades.OrmTransaction(bound, func(tx orm.Query) error {
		return tx.Table("configs").Create(configMarkerRow(key, "committed"))
	})
	require.NoError(t, err)

	var afterCommit int64
	err = conn.WithTenantConnection(tn, func() error {
		var qErr error
		afterCommit, qErr = countConfigKey(bound, key)
		return qErr
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), afterCommit, "tenant successful transaction must commit")

	if appfacades.SchemaHasTable(context.Background(), "configs") {
		platformN, pErr := appfacades.PlatformOrmQuery(nil).Table("configs").Where("key", key).Count()
		require.NoError(t, pErr)
		assert.Equal(t, int64(0), platformN, "tenant commit must not write platform configs")
	}
}