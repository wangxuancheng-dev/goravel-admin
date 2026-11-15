package feature_test

import (
	"os"
	"testing"

	"github.com/goravel/framework/facades"
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
