package feature_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/search"
	"goravel/app/services"
	"goravel/app/tenancy"
	"goravel/app/tenancyctx"
)

func TestDualTenantDatabaseIsolation(t *testing.T) {
	withTenancyDriver(t, "database")
	if !tenancy.Enabled() {
		t.Fatal("tenancy driver not enabled for test")
	}

	suffix := fmt.Sprintf("%d", time.Now().Unix()%1000000)
	codeA := "isoa" + suffix
	codeB := "isob" + suffix

	admin := services.NewTenantAdminService()
	conn := services.NewTenantConnectionService()

	ta, err := admin.Create(services.TenantCreateInput{
		Code:    codeA,
		Name:    "Isolation A",
		Migrate: false,
	})
	if err != nil {
		t.Skipf("skip dual-tenant isolation: cannot create tenant DB (need CREATE privilege): %v", err)
	}
	tb, err := admin.Create(services.TenantCreateInput{
		Code:    codeB,
		Name:    "Isolation B",
		Migrate: false,
	})
	if err != nil {
		_ = conn.DropStorage(ta)
		_, _ = appfacades.PlatformOrmQuery(nil).Where("id", ta.ID).Delete(&models.Tenant{})
		t.Skipf("skip dual-tenant isolation: cannot create second tenant DB: %v", err)
	}

	t.Cleanup(func() {
		_ = conn.DropStorage(ta)
		_ = conn.DropStorage(tb)
		_, _ = appfacades.PlatformOrmQuery(nil).Where("id", ta.ID).ForceDelete(&models.Tenant{})
		_, _ = appfacades.PlatformOrmQuery(nil).Where("id", tb.ID).ForceDelete(&models.Tenant{})
	})

	require.NoError(t, conn.MigrateTenant(ta))
	require.NoError(t, conn.MigrateTenant(tb))

	markerKey := "iso_marker_" + suffix
	err = conn.WithTenantConnection(ta, func() error {
		bound := tenancyctx.WithTenant(context.Background(), ta.ID, ta.ConnectionName, ta.Code)
		return appfacades.OrmQuery(bound).Table("configs").Create(map[string]any{
			"key":    markerKey,
			"value":  "from-tenant-a",
			"group":  "test",
			"type":   "string",
			"remark": "dual tenant isolation",
		})
	})
	require.NoError(t, err)

	var foundInA int64
	err = conn.WithTenantConnection(ta, func() error {
		bound := tenancyctx.WithTenant(context.Background(), ta.ID, ta.ConnectionName, ta.Code)
		var n int64
		n, err = appfacades.OrmQuery(bound).Table("configs").Where("key", markerKey).Count()
		foundInA = n
		return err
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), foundInA, "marker should exist in tenant A")

	var foundInB int64
	err = conn.WithTenantConnection(tb, func() error {
		bound := tenancyctx.WithTenant(context.Background(), tb.ID, tb.ConnectionName, tb.Code)
		var n int64
		n, err = appfacades.OrmQuery(bound).Table("configs").Where("key", markerKey).Count()
		foundInB = n
		return err
	})
	require.NoError(t, err)
	assert.Equal(t, int64(0), foundInB, "marker must not leak into tenant B")

	idxA := search.OrdersIndexShortNameFor(tenantCtx(ta))
	idxB := search.OrdersIndexShortNameFor(tenantCtx(tb))
	assert.True(t, strings.HasPrefix(idxA, ta.Code+"_"))
	assert.True(t, strings.HasPrefix(idxB, tb.Code+"_"))
	assert.NotEqual(t, idxA, idxB)
	assert.NotEqual(t, tenancy.StoragePrefix(tenantCtx(ta)), tenancy.StoragePrefix(tenantCtx(tb)))
}

func tenantCtx(t *models.Tenant) context.Context {
	return tenancyctx.WithTenant(context.Background(), t.ID, t.ConnectionName, t.Code)
}
