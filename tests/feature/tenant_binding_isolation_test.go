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

type dbView struct {
	ormDB, ormQueryDB, schemaConn, defaultConn string
}

// observeDB reports which database the process-wide (unbound) facades hit right now.
func observeDB(t *testing.T) dbView {
	t.Helper()
	var v dbView
	var err error
	v.ormDB, err = currentDatabase(facades.Orm().Query())
	require.NoError(t, err)
	v.ormQueryDB, err = currentDatabase(appfacades.OrmQuery(context.Background()))
	require.NoError(t, err)
	v.schemaConn = facades.Schema().GetConnection()
	v.defaultConn = facades.Config().GetString("database.default")
	return v
}

// While one goroutine works inside a tenant scope (WithTenantConnection / WithSchemaContext),
// other goroutines' unbound queries must keep hitting the platform database.
// The previous implementation flipped the root Orm query, the global Schema and
// database.default for the duration, leaking concurrent requests into the tenant DB.
func TestTenantScopeDoesNotLeakToOtherGoroutines(t *testing.T) {
	withTenancyDriver(t, "database")
	allowSharedTenantDBCredentials(t)
	if !tenancy.Enabled() {
		t.Fatal("tenancy driver not enabled for test")
	}
	require.True(t, appfacades.TenantRoutersInstalled(), "tenant-aware routers must be installed at boot")

	admin := services.NewTenantAdminService()
	conn := services.NewTenantConnectionService()
	code := fmt.Sprintf("bind%d", time.Now().UnixNano()%1000000)
	ta, err := admin.Create(services.TenantCreateInput{Code: code, Name: "Binding isolation", Migrate: false})
	if err != nil {
		skipWithoutTenantDB(t, err)
	}
	t.Cleanup(func() {
		_ = conn.DropStorage(ta)
		_, _ = appfacades.PlatformOrmQuery(nil).Where("id", ta.ID).ForceDelete(&models.Tenant{})
	})
	require.NoError(t, conn.MigrateTenant(ta))

	before := observeDB(t)
	require.NotEqual(t, ta.Database, before.ormDB, "precondition: root Orm is on the platform DB")

	// Run scope on this goroutine; observe from another one while it is active.
	runScoped := func(name string, scope func(inner func() error) error) {
		t.Run(name, func(t *testing.T) {
			var inside string
			var outside dbView
			err := scope(func() error {
				// Bound goroutine: facades route to the tenant.
				var err error
				if inside, err = currentDatabase(facades.Orm().Query()); err != nil {
					return err
				}
				done := make(chan struct{})
				go func() {
					defer close(done)
					outside = observeDB(t)
				}()
				<-done
				return nil
			})
			require.NoError(t, err)

			assert.Equal(t, ta.Database, inside, "inside the scope facades.Orm() must hit the tenant DB")
			assert.Equal(t, before, outside, "other goroutines must not observe the tenant scope")
			assert.Equal(t, before, observeDB(t), "nothing may be left switched after the scope")
		})
	}

	runScoped("WithTenantConnection", func(inner func() error) error {
		return conn.WithTenantConnection(ta, inner)
	})
	runScoped("WithSchemaContext", func(inner func() error) error {
		bound := tenancyctx.WithTenant(context.Background(), ta.ID, ta.ConnectionName, ta.Code)
		return appfacades.WithSchemaContext(bound, inner)
	})
	runScoped("nested", func(inner func() error) error {
		return conn.WithTenantConnection(ta, func() error {
			bound := tenancyctx.WithTenant(context.Background(), ta.ID, ta.ConnectionName, ta.Code)
			if !appfacades.SchemaHasTable(bound, "admins") {
				return fmt.Errorf("admins missing on tenant via nested SchemaHasTable")
			}
			return inner()
		})
	})
}
