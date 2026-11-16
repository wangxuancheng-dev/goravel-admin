package feature_test

import (
	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/services"
	"os"
	"testing"

	"github.com/goravel/framework/contracts/database/orm"
	"github.com/goravel/framework/facades"
	goravelpostgres "github.com/goravel/postgres"
)

// allowSharedTenantDBCredentials lets tenants on the platform DB host reuse the
// platform account, as the tenant tests do. .env.example (and so CI) keeps the
// public default false, under which creating such a tenant is rejected.
func allowSharedTenantDBCredentials(t *testing.T) {
	t.Helper()
	const key = "tenancy.allow_platform_db_credentials"
	prev := facades.Config().GetBool(key, false)
	facades.Config().Add(key, true)
	t.Cleanup(func() { facades.Config().Add(key, prev) })
}

// isPostgres reports whether q runs on PostgreSQL (Query.Driver() is the Goravel
// driver's name).
func isPostgres(q orm.Query) bool {
	return q != nil && q.Driver() == goravelpostgres.Name
}

// currentDatabase returns the database the server says q is connected to (not the
// configured name), to prove where a query was routed.
func currentDatabase(q orm.Query) (string, error) {
	sql := "SELECT DATABASE()"
	if isPostgres(q) {
		sql = "SELECT current_database()"
	}
	var name string
	err := q.Raw(sql).Scan(&name)
	return name, err
}

// skipWithoutTenantDB skips a test that cannot create a tenant database locally
// (no CREATE privilege), but fails it in CI, where the database is set up for it:
// a skipped tenant test in CI would silently stop covering anything.
func skipWithoutTenantDB(t *testing.T, err error) {
	t.Helper()
	if os.Getenv("CI") != "" {
		t.Fatalf("cannot create tenant DB in CI: %v", err)
	}
	t.Skipf("skip: cannot create tenant DB (need CREATE privilege): %v", err)
}

// withTenantSchema runs fn with tn's Schema bound to this goroutine (facades.Orm() /
// facades.Schema() hit the tenant), the way migrate / seed / the scope runner do.
func withTenantSchema(conn *services.TenantConnectionService, tn *models.Tenant, fn func() error) error {
	ctx, err := conn.TenantContext(tn)
	if err != nil {
		return err
	}
	return appfacades.WithSchemaContext(ctx, fn)
}
