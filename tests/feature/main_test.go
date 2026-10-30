package feature_test

import (
	"os"
	"testing"

	"github.com/goravel/framework/facades"

	_ "goravel/tests"
)

// TestMain runs feature tests in single-database mode by default, matching CI (.env.example),
// so a local .env with TENANCY_DRIVER=database does not break single-tenant tests.
// Tests that need tenancy opt in with withTenancyDriver(t, "database").
func TestMain(m *testing.M) {
	facades.Config().Add("tenancy.driver", "off")
	os.Exit(m.Run())
}
