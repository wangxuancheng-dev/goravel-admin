package feature_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/goravel/framework/facades"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/app/tenancyctx"
)

// TestTenantBackupOnDisk runs the backup lifecycle with TENANT_BACKUP_DISK set. The
// framework "local" disk stands in for object storage: the same disk API (put, list,
// get, delete) is used, the files just land under storage/app.
func TestTenantBackupOnDisk(t *testing.T) {
	withTenancyDriver(t, "database")
	allowSharedTenantDBCredentials(t)
	if !isPostgres(appfacades.PlatformQuery()) {
		for _, tool := range []string{"mysqldump", "mysql"} {
			if _, err := exec.LookPath(tool); err != nil {
				if os.Getenv("CI") != "" {
					t.Fatalf("%s not on PATH in CI", tool)
				}
				t.Skipf("skip: %s not on PATH", tool)
			}
		}
	} else {
		for _, tool := range []string{"pg_dump", "psql"} {
			if _, err := exec.LookPath(tool); err != nil {
				if os.Getenv("CI") != "" {
					t.Fatalf("%s not on PATH in CI", tool)
				}
				t.Skipf("skip: %s not on PATH", tool)
			}
		}
	}

	const key = "tenancy.backup_disk"
	prev := facades.Config().GetString(key, "")
	facades.Config().Add(key, "local")
	t.Cleanup(func() { facades.Config().Add(key, prev) })

	conn := services.NewTenantConnectionService()
	code := fmt.Sprintf("bkd%d", time.Now().UnixNano()%1_000_000_000)
	tn, err := services.NewTenantAdminService().Create(services.TenantCreateInput{Code: code, Name: "backup disk"})
	if err != nil {
		skipWithoutTenantDB(t, err)
	}
	t.Cleanup(func() {
		_ = services.PurgeTenantBackups(tn.Code)
		_ = conn.DropStorage(tn)
		_, _ = appfacades.PlatformQuery().Where("id", tn.ID).ForceDelete(&models.Tenant{})
	})
	require.NoError(t, conn.MigrateTenant(tn))

	bound := tenancyctx.WithTenant(context.Background(), tn.ID, tn.ConnectionName, tn.Code)
	setMarker := func(k string) {
		require.NoError(t, conn.WithTenantConnection(tn, func() error {
			return appfacades.OrmQuery(bound).Table("configs").Create(map[string]any{
				"key": k, "value": "x", "group": "test", "type": "string", "remark": "backup disk",
			})
		}))
	}
	hasMarker := func(k string) bool {
		var n int64
		require.NoError(t, conn.WithTenantConnection(tn, func() error {
			var countErr error
			n, countErr = appfacades.OrmQuery(bound).Table("configs").Where("key", k).Count()
			return countErr
		}))
		return n > 0
	}

	setMarker("before_backup")
	path, err := services.BackupTenant(tn, 0)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(path, "tenant-backups/"+tn.Code+"/"), "kept on the disk: %s", path)
	local, _ := filepath.Glob(filepath.Join(services.TenantBackupDir(tn.Code), "*.sql"))
	assert.Empty(t, local, "no local copy is left on this host")

	files, location, err := services.ListTenantBackups(tn)
	require.NoError(t, err)
	require.Len(t, files, 1)
	assert.Equal(t, "local:tenant-backups/"+tn.Code, location)
	assert.Greater(t, files[0].Size, int64(0))

	data, err := services.ReadTenantBackup(tn, files[0].Name)
	require.NoError(t, err)
	assert.NotEmpty(t, data, "the download endpoint reads the dump from the disk")
	_, err = services.ReadTenantBackup(tn, "../../etc/passwd.sql")
	assert.Error(t, err, "names cannot leave the tenant's backup dir")

	// Restore from the disk: the change made after the backup is gone, the temp
	// download is removed.
	setMarker("after_backup")
	tmp, cleanup, err := services.FetchTenantBackup(tn, files[0].Name)
	require.NoError(t, err)
	require.NoError(t, services.RestoreTenantFromFile(tn, tmp))
	cleanup()
	_, statErr := os.Stat(tmp)
	assert.True(t, os.IsNotExist(statErr), "the downloaded dump is removed after the restore")
	assert.True(t, hasMarker("before_backup"))
	assert.False(t, hasMarker("after_backup"), "restored to the backup")

	// Pruning keeps the newest dumps on the disk.
	time.Sleep(1100 * time.Millisecond) // dump names carry the second
	_, err = services.BackupTenant(tn, 0)
	require.NoError(t, err)
	removed, err := services.PruneTenantBackups(tn, 1)
	require.NoError(t, err)
	assert.Equal(t, 1, removed)
	files, _, err = services.ListTenantBackups(tn)
	require.NoError(t, err)
	assert.Len(t, files, 1)

	require.NoError(t, services.PurgeTenantBackups(tn.Code))
	files, _, err = services.ListTenantBackups(tn)
	require.NoError(t, err)
	assert.Empty(t, files, "purging a tenant removes its backups from the disk")
}
