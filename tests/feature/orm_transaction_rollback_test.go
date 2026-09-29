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

func countConfigKey(ctx context.Context, key string) (int64, error) {
	return appfacades.OrmQuery(ctx).Table("configs").Where("key", key).Count()
}

// TestOrmTransactionRollbackDefault covers single-merchant / default connection:
// insert inside OrmTransaction then return error must leave no row.
func TestOrmTransactionRollbackDefault(t *testing.T) {
	withTenancyDriver(t, "off")
	require.False(t, tenancy.Enabled())

	ctx := context.Background()
	if !appfacades.SchemaHasTable(ctx, "configs") {
		t.Skip("configs table missing on default connection")
	}

	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1_000_000_000)
	key := "tx_rb_def_" + suffix

	t.Cleanup(func() {
		_, _ = appfacades.OrmQuery(ctx).Table("configs").Where("key", key).Delete(&models.Config{})
	})

	err := appfacades.OrmTransaction(ctx, func(tx orm.Query) error {
		if err := tx.Table("configs").Create(configMarkerRow(key, "should-roll-back")); err != nil {
			return err
		}
		return errors.New("force rollback")
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "force rollback")

	n, err := countConfigKey(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, int64(0), n, "failed transaction must not leave a configs row")

	err = appfacades.OrmTransaction(ctx, func(tx orm.Query) error {
		return tx.Table("configs").Create(configMarkerRow(key, "committed"))
	})
	require.NoError(t, err)

	n, err = countConfigKey(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, int64(1), n, "successful transaction must commit")
}

// TestOrmTransactionRollbackTenant covers multi-tenant: OrmTransaction must open
// on the tenant connection; rollback must not leave data in that tenant DB.
func TestOrmTransactionRollbackTenant(t *testing.T) {
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
	code := "txrb" + suffix
	key := "tx_rb_ten_" + suffix

	adminSvc := services.NewTenantAdminService()
	conn := services.NewTenantConnectionService()

	tn, err := adminSvc.Create(services.TenantCreateInput{
		Code:    code,
		Name:    "TX Rollback",
		Migrate: true,
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

	// Platform / default must not receive the marker (isolation).
	if appfacades.SchemaHasTable(context.Background(), "configs") {
		platformN, pErr := appfacades.PlatformOrmQuery(nil).Table("configs").Where("key", key).Count()
		require.NoError(t, pErr)
		assert.Equal(t, int64(0), platformN, "tenant commit must not write platform configs")
	}
}
