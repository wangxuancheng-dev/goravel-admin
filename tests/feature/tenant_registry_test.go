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
	"goravel/app/tenantdb"
)

// Tenant connections are owned by tenantdb: registering one must not write the global
// config, Forget must really close the pool, and re-registering afterwards must give a
// working pool (the framework's driver cache used to hand back the closed one).
func TestTenantRegistryOwnsConnections(t *testing.T) {
	withTenancyDriver(t, "database")
	allowSharedTenantDBCredentials(t)
	if !tenancy.Enabled() {
		t.Fatal("tenancy driver not enabled for test")
	}

	admin := services.NewTenantAdminService()
	conn := services.NewTenantConnectionService()
	code := fmt.Sprintf("reg%d", time.Now().UnixNano()%1000000)
	ta, err := admin.Create(services.TenantCreateInput{Code: code, Name: "Registry", Migrate: false})
	if err != nil {
		skipWithoutTenantDB(t, err)
	}
	t.Cleanup(func() {
		_ = conn.DropStorage(ta)
		_, _ = appfacades.PlatformOrmQuery(nil).Where("id", ta.ID).ForceDelete(&models.Tenant{})
	})
	require.NoError(t, conn.MigrateTenant(ta))

	require.NoError(t, conn.EnsureRegistered(ta))
	assert.Nil(t, facades.Config().Get("database.connections."+ta.ConnectionName),
		"tenant connections must not be written into the global config")
	assert.Equal(t, ta.Database, appfacades.ConnectionDatabaseName(ta.ConnectionName))

	first, ok := tenantdb.Default.Get(ta.ConnectionName)
	require.True(t, ok, "EnsureRegistered must register the connection")

	bound := tenancyctx.WithTenant(context.Background(), ta.ID, ta.ConnectionName, ta.Code)
	countAdmins := func() int64 {
		t.Helper()
		dbName, err := currentDatabase(appfacades.OrmQuery(bound))
		require.NoError(t, err)
		require.Equal(t, ta.Database, dbName, "OrmQuery(tenant ctx) must hit the tenant database")
		n, err := appfacades.OrmQuery(bound).Table("admins").Count()
		require.NoError(t, err)
		return n
	}
	countAdmins()

	// Forget closes the pool now.
	conn.Forget(ta.ConnectionName)
	_, still := tenantdb.Default.Get(ta.ConnectionName)
	assert.False(t, still)
	assert.Error(t, first.DB().Ping(), "Forget must close the pool")

	// Same connection name again: a new, working pool (was: the closed cached one).
	require.NoError(t, conn.EnsureRegistered(ta))
	second, ok := tenantdb.Default.Get(ta.ConnectionName)
	require.True(t, ok)
	assert.NotSame(t, first, second)
	countAdmins()

	// An evicted connection referenced by a ctx is rebuilt on demand from the tenants table.
	conn.Forget(ta.ConnectionName)
	countAdmins()
	_, rebuilt := tenantdb.Default.Get(ta.ConnectionName)
	assert.True(t, rebuilt, "OrmQuery must rebuild an evicted tenant connection")

	// Schema helpers resolve the tenant through the registry too.
	assert.True(t, appfacades.SchemaHasTable(bound, "admins"))
	assert.False(t, appfacades.SchemaHasTable(bound, "tenants"), "tenants is a platform table")
}
