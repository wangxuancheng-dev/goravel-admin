package feature_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/goravel/framework/facades"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/services"
)

// TENANCY_DB_DEDICATED_USER: the tenant logs in with an account of its own, which
// the database server keeps out of the platform database whatever the code asks for;
// the account goes with the tenant's storage.
func TestDedicatedTenantDBUserIsolatedByServer(t *testing.T) {
	withTenancyDriver(t, "database")
	prev := facades.Config().GetBool("tenancy.db_dedicated_user", false)
	facades.Config().Add("tenancy.db_dedicated_user", true)
	t.Cleanup(func() { facades.Config().Add("tenancy.db_dedicated_user", prev) })

	code := fmt.Sprintf("dedu%d", time.Now().UnixNano()%1000000)
	admin := services.NewTenantAdminService()
	conn := services.NewTenantConnectionService()
	tn, err := admin.Create(services.TenantCreateInput{Code: code, Name: "Dedicated", Migrate: false})
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "denied") {
			t.Skipf("platform DB account cannot create users: %v", err)
		}
		skipWithoutTenantDB(t, err)
	}
	dropped := false
	t.Cleanup(func() {
		if !dropped {
			_ = conn.DropStorage(tn)
		}
		_, _ = appfacades.PlatformQuery().Where("id", tn.ID).ForceDelete(&models.Tenant{})
	})
	assert.Equal(t, services.DedicatedTenantDBUsername(code), tn.Username)
	require.NoError(t, conn.MigrateTenant(tn))

	// The generated password is never written to the SQL log / Goralens tables.
	password, err := services.RevealTenantPassword(tn.Password)
	require.NoError(t, err)
	if appfacades.SchemaHasTable(context.Background(), "goralens_entries") {
		time.Sleep(600 * time.Millisecond) // Goralens flushes in batches
		n, countErr := appfacades.PlatformQuery().Table("goralens_entries").Where("content LIKE ?", "%"+password+"%").Count()
		require.NoError(t, countErr)
		assert.Zero(t, n, "the tenant DB password was logged")
	}

	// An operator cannot claim the reserved account name for another tenant setup.
	_, err = admin.Create(services.TenantCreateInput{Code: code + "x", Name: "x", Username: services.DedicatedTenantDBUsername(code + "x"), Password: "p"})
	assert.ErrorContains(t, err, "reserved")

	ctx, err := conn.TenantContext(tn)
	require.NoError(t, err)
	q := appfacades.OrmQuery(ctx)
	postgres := isPostgres(q)

	// It is the dedicated account that is logged in.
	var who []struct {
		U string `gorm:"column:u"`
	}
	whoSQL := "SELECT CURRENT_USER() AS u"
	if postgres {
		whoSQL = "SELECT current_user AS u"
	}
	require.NoError(t, appfacades.OrmQuery(ctx).Raw(whoSQL).Scan(&who))
	require.Len(t, who, 1)
	assert.True(t, strings.HasPrefix(who[0].U, tn.Username), "current user %q", who[0].U)

	// Its own database works.
	_, err = appfacades.OrmQuery(ctx).Table("admins").Count()
	require.NoError(t, err)

	// Raw SQL at the platform database is refused by the server itself.
	platformDB, err := currentDatabase(appfacades.PlatformQuery())
	require.NoError(t, err)
	var n int64
	if postgres {
		// Another database is out of reach on PostgreSQL anyway; the platform's tables
		// in a shared database are what the role must not read.
		err = appfacades.OrmQuery(ctx).Raw(`SELECT COUNT(*) FROM public.tenants`).Scan(&n)
	} else {
		err = appfacades.OrmQuery(ctx).Raw(fmt.Sprintf("SELECT COUNT(*) FROM `%s`.`tenants`", platformDB)).Scan(&n)
	}
	require.Error(t, err, "the dedicated account read the platform database")

	// Backup and restore run with the dedicated account (no PROCESS / SUPER).
	if _, err := exec.LookPath(map[bool]string{true: "pg_dump", false: "mysqldump"}[postgres]); err == nil {
		_, err := services.BackupTenant(tn, 0)
		require.NoError(t, err, "backup with the dedicated account")
		files, _, err := services.ListTenantBackups(tn)
		require.NoError(t, err)
		require.NotEmpty(t, files)
		tmp, cleanup, err := services.FetchTenantBackup(tn, files[0].Name)
		require.NoError(t, err)
		require.NoError(t, services.RestoreTenantFromFile(tn, tmp), "restore with the dedicated account")
		cleanup()
		_ = os.RemoveAll(services.TenantBackupDir(tn.Code))
	}

	// Dropping the storage drops the account.
	require.NoError(t, conn.DropStorage(tn))
	dropped = true
	var rows []struct {
		N int64 `gorm:"column:n"`
	}
	countSQL := "SELECT COUNT(*) AS n FROM mysql.user WHERE user = ?"
	if postgres {
		countSQL = "SELECT COUNT(*) AS n FROM pg_roles WHERE rolname = ?"
	}
	require.NoError(t, appfacades.PlatformQuery().Raw(countSQL, tn.Username).Scan(&rows))
	require.Len(t, rows, 1)
	assert.Zero(t, rows[0].N, "account %s left behind", tn.Username)
}

// A tenant without credentials of its own runs as the platform account only while
// TENANCY_ALLOW_PLATFORM_DB_CREDENTIALS=true: turning it off stops it at once, not
// only for tenants created afterwards.
func TestPlatformCredentialFallbackFollowsSetting(t *testing.T) {
	withTenancyDriver(t, "database")
	allowSharedTenantDBCredentials(t)

	code := fmt.Sprintf("shared%d", time.Now().UnixNano()%1000000)
	admin := services.NewTenantAdminService()
	conn := services.NewTenantConnectionService()
	tn, err := admin.Create(services.TenantCreateInput{Code: code, Name: "Shared", Migrate: false})
	if err != nil {
		skipWithoutTenantDB(t, err)
	}
	t.Cleanup(func() {
		facades.Config().Add("tenancy.allow_platform_db_credentials", true)
		_ = conn.DropStorage(tn)
		_, _ = appfacades.PlatformQuery().Where("id", tn.ID).ForceDelete(&models.Tenant{})
	})
	require.Empty(t, tn.Username, "precondition: no credentials of its own")
	_, err = conn.TenantContext(tn)
	require.NoError(t, err)

	facades.Config().Add("tenancy.allow_platform_db_credentials", false)
	conn.Forget(tn.ConnectionName)
	_, err = conn.TenantContext(tn)
	assert.ErrorContains(t, err, apperrors.ErrTenantCredentialsRequired.Code)
	_, _, _, _, _, err = services.ResolveTenantDSN(tn)
	assert.ErrorContains(t, err, apperrors.ErrTenantCredentialsRequired.Code, "backups must not fall back either")
}

// What a permanent delete left behind before deletes erased storage (database and
// dedicated account, row gone) is found by tenant:orphans and dropped; storage of a
// live tenant is never listed.
func TestOrphanTenantStorageFoundAndDropped(t *testing.T) {
	withTenancyDriver(t, "database")
	prev := facades.Config().GetBool("tenancy.db_dedicated_user", false)
	facades.Config().Add("tenancy.db_dedicated_user", true)
	t.Cleanup(func() { facades.Config().Add("tenancy.db_dedicated_user", prev) })

	admin := services.NewTenantAdminService()
	conn := services.NewTenantConnectionService()
	suffix := fmt.Sprint(time.Now().UnixNano() % 1000000)
	live, err := admin.Create(services.TenantCreateInput{Code: "live" + suffix, Name: "live"})
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "denied") {
			t.Skipf("platform DB account cannot create users: %v", err)
		}
		skipWithoutTenantDB(t, err)
	}
	gone, err := admin.Create(services.TenantCreateInput{Code: "gone" + suffix, Name: "gone"})
	require.NoError(t, err)
	t.Cleanup(func() {
		for _, tn := range []*models.Tenant{live, gone} {
			_ = conn.DropStorage(tn)
			_, _ = appfacades.PlatformQuery().WithTrashed().Where("id", tn.ID).ForceDelete(&models.Tenant{})
		}
	})
	// The old way: the row goes, the database and the account stay.
	conn.Forget(gone.ConnectionName)
	_, err = appfacades.PlatformQuery().WithTrashed().Where("id", gone.ID).ForceDelete(&models.Tenant{})
	require.NoError(t, err)

	orphans, _, err := services.FindOrphanTenantStorage()
	require.NoError(t, err)
	var found *services.OrphanTenantStorage
	for i := range orphans {
		assert.NotEqual(t, live.Database, orphans[i].Name, "a live tenant's database is never an orphan")
		if orphans[i].Name == gone.Database {
			found = &orphans[i]
		}
	}
	require.NotNil(t, found, "the leftover database %s must be listed", gone.Database)
	assert.Equal(t, gone.Code, found.Code)
	assert.Equal(t, gone.Username, found.Account, "its dedicated account is paired with it")

	require.NoError(t, services.DropOrphanTenantStorage(*found))
	exists, err := tenantDatabaseExists(gone.Database)
	require.NoError(t, err)
	assert.False(t, exists)
	orphans, loose, err := services.FindOrphanTenantStorage()
	require.NoError(t, err)
	for _, o := range orphans {
		assert.NotEqual(t, gone.Database, o.Name)
	}
	assert.NotContains(t, loose, gone.Username, "the account went with the database")

	// The code is usable again.
	again, err := admin.Create(services.TenantCreateInput{Code: gone.Code, Name: "again"})
	require.NoError(t, err)
	gone.ID = again.ID
}

// Changing a dedicated account's password from the platform changes it on the server
// too (ALTER USER), so the tenant keeps working with the new one.
func TestDedicatedTenantDBPasswordChange(t *testing.T) {
	withTenancyDriver(t, "database")
	prev := facades.Config().GetBool("tenancy.db_dedicated_user", false)
	facades.Config().Add("tenancy.db_dedicated_user", true)
	t.Cleanup(func() { facades.Config().Add("tenancy.db_dedicated_user", prev) })

	admin := services.NewTenantAdminService()
	conn := services.NewTenantConnectionService()
	code := fmt.Sprintf("pwchg%d", time.Now().UnixNano()%1000000)
	tn, err := admin.Create(services.TenantCreateInput{Code: code, Name: code})
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "denied") {
			t.Skipf("platform DB account cannot create users: %v", err)
		}
		skipWithoutTenantDB(t, err)
	}
	t.Cleanup(func() {
		_ = conn.DropStorage(tn)
		_, _ = appfacades.PlatformQuery().WithTrashed().Where("id", tn.ID).ForceDelete(&models.Tenant{})
	})
	require.NoError(t, conn.MigrateTenant(tn))

	newPassword := "Rotated" + code + "X9"
	updated, err := admin.UpdateConnection(tn.ID, services.TenantUpdateInput{Password: &newPassword})
	require.NoError(t, err)
	plain, err := services.RevealTenantPassword(updated.Password)
	require.NoError(t, err)
	require.Equal(t, newPassword, plain)

	ctx, err := conn.TenantContext(updated)
	require.NoError(t, err, "the tenant connects with the new password")
	_, err = appfacades.OrmQuery(ctx).Table("admins").Count()
	require.NoError(t, err)

	bad := "not-alnum!"
	_, err = admin.UpdateConnection(tn.ID, services.TenantUpdateInput{Password: &bad})
	assert.Error(t, err, "dedicated passwords must stay alphanumeric")
}
