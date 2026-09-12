package feature_test

import (
	"context"
	"testing"

	"github.com/goravel/framework/facades"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "goravel/app/errors"
	"goravel/app/jobs"
	"goravel/app/queuejobs"
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

	job := &queuejobs.SyncOrderSearch{}
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
