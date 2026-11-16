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
