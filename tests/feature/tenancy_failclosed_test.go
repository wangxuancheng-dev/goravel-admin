package feature_test

import (
	"context"
	"testing"

	"github.com/goravel/framework/contracts/database/orm"
	"github.com/goravel/framework/facades"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/jobs"
	"goravel/app/search"
	"goravel/app/tenancy"
)

func withSearchOrdersSync(t *testing.T, enabled bool) {
	t.Helper()
	prevEnabled := facades.Config().GetBool("search.enabled", false)
	prevDriver := facades.Config().GetString("search.driver", "null")
	prevSync := facades.Config().GetBool("search.indexes.orders.sync_enabled", false)
	if enabled {
		facades.Config().Add("search.enabled", true)
		facades.Config().Add("search.driver", "elasticsearch")
		facades.Config().Add("search.indexes.orders.sync_enabled", true)
	} else {
		facades.Config().Add("search.enabled", false)
	}
	t.Cleanup(func() {
		facades.Config().Add("search.enabled", prevEnabled)
		facades.Config().Add("search.driver", prevDriver)
		facades.Config().Add("search.indexes.orders.sync_enabled", prevSync)
	})
}

func TestExportJobContextFailClosedWithoutTenant(t *testing.T) {
	withTenancyDriver(t, "database")
	require.True(t, tenancy.Enabled())

	_, err := jobs.JobContext(jobs.ExportArgs{ExportID: 1, AdminID: 1})
	require.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrTenantRequired)
}

func TestExportJobContextAllowsWhenTenancyOff(t *testing.T) {
	withTenancyDriver(t, "off")
	require.False(t, tenancy.Enabled())

	ctx, err := jobs.JobContext(jobs.ExportArgs{ExportID: 1, AdminID: 1})
	require.NoError(t, err)
	require.NotNil(t, ctx)
}

func TestSyncOrderSearchFailClosedWithoutTenant(t *testing.T) {
	withTenancyDriver(t, "database")
	withSearchOrdersSync(t, true)
	require.True(t, tenancy.Enabled())
	require.True(t, search.OrdersSyncEnabled())

	job := &jobs.SyncOrderSearch{}
	err := job.Handle(map[string]any{
		"order_id": uint(1),
		"op":       "index",
		"order_no": "ORD-1",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrTenantRequired)
}

func TestOrdersIndexShortNameFailClosedUnbound(t *testing.T) {
	withTenancyDriver(t, "database")
	require.True(t, tenancy.Enabled())
	assert.Equal(t, "", search.OrdersIndexShortNameFor(context.Background()))
}

func TestCacheAndStorageUnboundDoNotShareRoot(t *testing.T) {
	withTenancyDriver(t, "database")
	require.True(t, tenancy.Enabled())
	assert.Equal(t, "t_unbound:lock:x", tenancy.CacheKey(context.Background(), "lock:x"))
	assert.Equal(t, "tenants/_unbound_/", tenancy.StoragePrefix(context.Background()))
}

// A forgotten (nil) or unbound ctx must not silently read or write the platform database.
//
//nolint:staticcheck // SA1012: passing nil is the point of the test.
func TestOrmQueryNilContextFailsClosedWithTenancy(t *testing.T) {
	withTenancyDriver(t, "database")
	require.True(t, tenancy.Enabled())

	_, err := appfacades.OrmQuery(nil).Table("admins").Count()
	require.Error(t, err)
	assert.ErrorIs(t, err, appfacades.ErrNoTenantContext)

	err = appfacades.OrmTransaction(nil, func(orm.Query) error { return nil })
	assert.ErrorIs(t, err, appfacades.ErrNoTenantContext)

	// A non-nil ctx bound to no tenant is the same mistake.
	_, err = appfacades.OrmQuery(context.Background()).Table("admins").Count()
	assert.ErrorIs(t, err, appfacades.ErrNoTenantContext)
	err = appfacades.OrmTransaction(context.Background(), func(orm.Query) error { return nil })
	assert.ErrorIs(t, err, appfacades.ErrNoTenantContext)

	// Platform code says so explicitly.
	_, err = appfacades.PlatformQuery().Table("tenants").Count()
	require.NoError(t, err)
}

//nolint:staticcheck // SA1012: passing nil is the point of the test.
func TestOrmQueryNilContextAllowedWhenTenancyOff(t *testing.T) {
	withTenancyDriver(t, "off")
	_, err := appfacades.OrmQuery(nil).Table("admins").Count()
	require.NoError(t, err)
}
