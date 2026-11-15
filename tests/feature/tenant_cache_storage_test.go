package feature_test

import (
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
	"goravel/app/tenantcache"
	"goravel/app/tenantstorage"
)

// newMetaTenant inserts a platform tenants row only (no tenant database), enough for
// tenant resolution and storage.
func newMetaTenant(t *testing.T, prefix string) *models.Tenant {
	t.Helper()
	code := fmt.Sprintf("%s%d", prefix, time.Now().UnixNano()%1000000)
	tn := &models.Tenant{
		Code:            code,
		Name:            code,
		Status:          models.TenantStatusActive,
		ProvisionStatus: models.TenantProvisionReady,
		Driver:          models.TenantDriverMySQL,
		Isolation:       models.TenantIsolationDatabase,
		Database:        "tenant_" + code,
		ConnectionName:  "tenant_meta_" + code,
		StorageMode:     models.TenantStorageModeShared,
	}
	require.NoError(t, appfacades.PlatformOrmQuery(nil).Create(tn))
	t.Cleanup(func() {
		_, _ = appfacades.PlatformOrmQuery(nil).WithTrashed().Where("id", tn.ID).ForceDelete(&models.Tenant{})
		tenantcache.Invalidate(tn.ID)
		tenantstorage.Invalidate(tn.ID)
	})
	return tn
}

// Tenant rows are read on every tenant request, so they are cached; every kind of
// write must show up at once (disabling a tenant must not wait for the cache), with no
// manual invalidation anywhere.
func TestTenantRowCacheInvalidatesOnWrites(t *testing.T) {
	withTenancyDriver(t, "database")
	if !tenancy.Enabled() {
		t.Fatal("tenancy driver not enabled for test")
	}
	conn := services.NewTenantConnectionService()
	tn := newMetaTenant(t, "tc")

	read := func() *models.Tenant {
		t.Helper()
		got, err := conn.FindTenantByIDOrCode(tn.Code)
		require.NoError(t, err)
		require.True(t, tenantcache.IsCached(tn.ID), "a read must leave the row cached")
		return got
	}

	// Callers get copies: mutating one must not leak into the cache.
	first := read()
	first.Status = models.TenantStatusDisabled
	assert.Equal(t, models.TenantStatusActive, read().Status)

	for _, w := range []struct {
		name  string
		write func() error
		check func(*models.Tenant) bool
	}{
		{"Model(t).Update", func() error {
			_, err := appfacades.PlatformOrmQuery(nil).Model(tn).Update(map[string]any{"status": models.TenantStatusDisabled})
			return err
		}, func(r *models.Tenant) bool { return r.Status == models.TenantStatusDisabled }},
		{"bulk Where(id).Update", func() error {
			_, err := appfacades.PlatformOrmQuery(nil).Model(&models.Tenant{}).Where("id", tn.ID).Update(map[string]any{"status": models.TenantStatusActive})
			return err
		}, func(r *models.Tenant) bool { return r.Status == models.TenantStatusActive }},
		{"raw Exec UPDATE", func() error {
			_, err := appfacades.PlatformOrmQuery(nil).Exec("UPDATE tenants SET maintenance = ? WHERE id = ?", true, tn.ID)
			return err
		}, func(r *models.Tenant) bool { return r.Maintenance }},
	} {
		read() // make sure the row is cached before the write
		require.NoError(t, w.write(), w.name)
		assert.False(t, tenantcache.IsCached(tn.ID), "%s must invalidate the cached row", w.name)
		assert.True(t, w.check(read()), "%s must be visible immediately", w.name)
	}

	// Soft delete through the platform service (a bulk Where(...).Delete).
	_, err := appfacades.PlatformOrmQuery(nil).Model(tn).Update(map[string]any{"maintenance": false})
	require.NoError(t, err)
	read()
	_, err = services.NewTenantAdminService().DeleteTenant(tn.ID, tn.Code, services.TenantDeleteOptions{})
	require.NoError(t, err)
	_, err = conn.FindTenantByIDOrCode(tn.Code)
	assert.Error(t, err, "a deleted tenant must stop resolving immediately")
	_, err = conn.FindTenantByIDOrCode(fmt.Sprint(tn.ID))
	assert.Error(t, err)
}

// A tenant's own bucket (BYOB) gets a driver built from its credentials: reused across
// opens, rebuilt when the credentials change, never written to the global config.
func TestTenantByobDiskDriverLifecycle(t *testing.T) {
	withTenancyDriver(t, "database")
	tn := newMetaTenant(t, "byob")

	sealed, err := tenantstorage.SealSecret("first-secret")
	require.NoError(t, err)
	_, err = appfacades.PlatformOrmQuery(nil).Model(tn).Update(map[string]any{
		"storage_mode":     models.TenantStorageModeCustom,
		"storage_driver":   "minio",
		"storage_key":      "key-1",
		"storage_secret":   sealed,
		"storage_bucket":   "bucket-1",
		"storage_url":      "https://files.example.com",
		"storage_endpoint": "minio.example.com:9000",
		"storage_ssl":      true,
	})
	require.NoError(t, err)

	disk := tenantstorage.DiskName(tn.ID)
	first, err := tenantstorage.OpenDisk(disk)
	require.NoError(t, err)
	second, err := tenantstorage.OpenDisk(disk)
	require.NoError(t, err)
	assert.Same(t, first, second, "the driver must be reused, not rebuilt per open")

	assert.Nil(t, facades.Config().Get("filesystems.disks."+disk),
		"tenant credentials must not be written to the global config")
	cfg, err := tenantstorage.DiskConfig(disk)
	require.NoError(t, err)
	assert.Equal(t, "key-1", cfg.GetString("filesystems.disks."+disk+".key"))
	assert.Equal(t, "first-secret", cfg.GetString("filesystems.disks."+disk+".secret"))

	// Rotating the credentials (ORM write -> row cache invalidated) gives a new driver.
	_, err = appfacades.PlatformOrmQuery(nil).Model(tn).Update(map[string]any{"storage_key": "key-2"})
	require.NoError(t, err)
	third, err := tenantstorage.OpenDisk(disk)
	require.NoError(t, err)
	assert.NotSame(t, first, third, "changed credentials must rebuild the driver")
	cfg, err = tenantstorage.DiskConfig(disk)
	require.NoError(t, err)
	assert.Equal(t, "key-2", cfg.GetString("filesystems.disks."+disk+".key"))
}

// The platform already lets an operator give one tenant its own bucket. Drive that
// through the real platform API: turning it on routes the tenant's new uploads to its
// bucket, rotating the key takes effect at once, and switching back restores the
// shared disk.
func TestPlatformTenantStorageSettingsOverHTTP(t *testing.T) {
	withTenancyDriver(t, "database")
	allowSharedTenantDBCredentials(t)
	token := loginPlatformSmoke(t)
	tn := newMetaTenant(t, "pstore")
	uri := fmt.Sprintf("/api/platform/tenants/%d", tn.ID)
	ctx := tenantCtx(tn)
	disk := tenantstorage.DiskName(tn.ID)

	require.Empty(t, tenantstorage.ResolveWriteDisk(ctx), "shared by default")

	resp, content := platformCall(t, token, "PUT", uri, `{
		"storage_mode": "custom", "storage_driver": "minio",
		"storage_key": "plat-key-1", "storage_secret": "plat-secret",
		"storage_bucket": "tenant-bucket", "storage_url": "https://files.example.com",
		"storage_endpoint": "minio.example.com:9000", "storage_ssl": true
	}`)
	require.True(t, resp.IsSuccessful(), content)
	assert.NotContains(t, content, "plat-secret", "the API must never echo the secret")

	assert.Equal(t, disk, tenantstorage.ResolveWriteDisk(ctx), "new uploads must go to the tenant's own bucket")
	driver, err := tenantstorage.OpenDisk(disk)
	require.NoError(t, err)
	require.NotNil(t, driver)
	cfg, err := tenantstorage.DiskConfig(disk)
	require.NoError(t, err)
	assert.Equal(t, "plat-key-1", cfg.GetString("filesystems.disks."+disk+".key"))
	assert.Equal(t, "plat-secret", cfg.GetString("filesystems.disks."+disk+".secret"), "secret is sealed at rest and revealed for the driver")
	assert.Nil(t, facades.Config().Get("filesystems.disks."+disk), "credentials stay out of the global config")

	resp, content = platformCall(t, token, "PUT", uri, `{"storage_key": "plat-key-2"}`)
	require.True(t, resp.IsSuccessful(), content)
	cfg, err = tenantstorage.DiskConfig(disk)
	require.NoError(t, err)
	assert.Equal(t, "plat-key-2", cfg.GetString("filesystems.disks."+disk+".key"), "a rotated key must apply immediately")

	resp, content = platformCall(t, token, "PUT", uri, `{"storage_mode": "shared"}`)
	require.True(t, resp.IsSuccessful(), content)
	assert.Empty(t, tenantstorage.ResolveWriteDisk(ctx), "switching back must restore the shared disk for new uploads")
	_, err = tenantstorage.OpenDisk(disk)
	assert.NoError(t, err, "files already in the tenant's bucket must stay readable after switching back")
}

// Orm().Fresh() rebuilds the framework's connections; the invalidation hooks live on
// the platform gorm instance, so they must still apply afterwards (or the cache must
// be off) — never a cache that silently stopped invalidating.
func TestTenantRowCacheStillInvalidatesAfterOrmFresh(t *testing.T) {
	withTenancyDriver(t, "database")
	conn := services.NewTenantConnectionService()
	tn := newMetaTenant(t, "tcf")

	facades.Orm().Fresh()

	_, err := conn.FindTenantByIDOrCode(tn.Code)
	require.NoError(t, err)
	require.True(t, tenantcache.IsCached(tn.ID), "the cache must be back on after Fresh")

	_, err = appfacades.PlatformOrmQuery(nil).Model(&models.Tenant{}).Where("id", tn.ID).Update(map[string]any{"status": models.TenantStatusDisabled})
	require.NoError(t, err)
	assert.False(t, tenantcache.IsCached(tn.ID), "writes after Fresh must still invalidate")
	got, err := conn.FindTenantByIDOrCode(tn.Code)
	require.NoError(t, err)
	assert.Equal(t, models.TenantStatusDisabled, got.Status)
}
