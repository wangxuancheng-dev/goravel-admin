package feature_test

import (
	"fmt"
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
	"goravel/tests"
)

// tenantGET calls GET uri on the tenant admin API with the given headers and returns
// the business code (HTTP-like 200 / 401 / ...) and the error code string, if any.
func tenantGET(t *testing.T, headers map[string]string, uri string) (int, string) {
	t.Helper()
	req := (&tests.TestCase{}).Http(t)
	for k, v := range headers {
		req = req.WithHeader(k, v)
	}
	resp, err := req.Get(uri)
	require.NoError(t, err)
	payload, err := resp.Json()
	require.NoError(t, err)
	code, _ := payload["code"].(float64)
	msg := fmt.Sprint(payload["error_code"], " ", payload["message"], " ", payload["error"])
	return int(code), msg
}

func withConfig(t *testing.T, key string, value any) {
	t.Helper()
	prev := facades.Config().Get(key)
	facades.Config().Add(key, value)
	t.Cleanup(func() { facades.Config().Add(key, prev) })
}

// Who decides which tenant a request belongs to, and what a token is good for: the
// ways a client could try to reach another tenant's data, or keep reaching its own
// after the platform shut it out.
func TestTenantResolutionSecurity(t *testing.T) {
	withTenancyDriver(t, "database")
	allowSharedTenantDBCredentials(t)

	admin := services.NewTenantAdminService()
	conn := services.NewTenantConnectionService()
	suffix := fmt.Sprint(time.Now().UnixNano() % 1000000)
	create := func(code string) *models.Tenant {
		tn, err := admin.Create(services.TenantCreateInput{Code: code, Name: code, Migrate: false})
		if err != nil {
			skipWithoutTenantDB(t, err)
		}
		t.Cleanup(func() {
			_ = conn.DropStorage(tn)
			_, _ = appfacades.PlatformQuery().WithTrashed().Where("id", tn.ID).ForceDelete(&models.Tenant{})
		})
		require.NoError(t, conn.MigrateTenant(tn))
		require.NoError(t, conn.SeedTenant(tn))
		return tn
	}
	ta, tb := create("seca"+suffix), create("secb"+suffix)
	tokenA := tenantAdminLogin(t, ta.Code)
	asA := func(extra map[string]string) map[string]string {
		h := map[string]string{"X-Tenant-ID": ta.Code, "Authorization": "Bearer " + tokenA}
		for k, v := range extra {
			h[k] = v
		}
		return h
	}
	code, _ := tenantGET(t, asA(nil), "/api/admin/info")
	require.Equal(t, 200, code)

	t.Run("platform and tenant tokens do not cross", func(t *testing.T) {
		platformToken := loginPlatformSmoke(t)
		code, _ := tenantGET(t, map[string]string{"X-Tenant-ID": ta.Code, "Authorization": "Bearer " + platformToken}, "/api/admin/info")
		assert.Equal(t, 401, code, "a platform token must not open a tenant API")
		resp, body := platformCall(t, tokenA, "GET", "/api/platform/tenants", "")
		assert.False(t, resp.IsSuccessful(), "a tenant token must not open the platform API: %s", body)
	})

	t.Run("host and header naming different tenants conflict", func(t *testing.T) {
		withConfig(t, "tenancy.resolver", "subdomain")
		withConfig(t, "tenancy.base_domain", "example.test")
		code, msg := tenantGET(t, asA(map[string]string{"X-Forwarded-Host": tb.Code + ".example.test"}), "/api/admin/info")
		assert.Equal(t, 400, code)
		assert.Contains(t, msg, apperrors.ErrTenantHintConflict.Code)
		// Same through the browser Origin (split SPA / API hosts).
		code, msg = tenantGET(t, asA(map[string]string{"Origin": "https://" + tb.Code + ".example.test"}), "/api/admin/info")
		assert.Equal(t, 400, code)
		assert.Contains(t, msg, apperrors.ErrTenantHintConflict.Code)
		// The host alone picks the tenant: A's token is not valid for B's host.
		code, _ = tenantGET(t, map[string]string{
			"X-Forwarded-Host": tb.Code + ".example.test", "Authorization": "Bearer " + tokenA,
		}, "/api/admin/info")
		assert.Equal(t, 401, code)
	})

	t.Run("public apex ignores a client-supplied tenant", func(t *testing.T) {
		withConfig(t, "tenancy.resolver", "subdomain")
		withConfig(t, "tenancy.base_domain", "example.test")
		withConfig(t, "tenancy.allow_header_fallback", "false")
		for _, host := range []string{"example.test", "www.example.test", "platform.example.test"} {
			code, msg := tenantGET(t, asA(map[string]string{"X-Forwarded-Host": host}), "/api/admin/info")
			assert.Equal(t, 400, code, host)
			assert.Contains(t, msg, apperrors.ErrTenantRequired.Code, host)
		}
		// API hosts accept it (split SPA / API), and the token still decides access.
		code, _ := tenantGET(t, asA(map[string]string{"X-Forwarded-Host": "api.example.test"}), "/api/admin/info")
		assert.Equal(t, 200, code)
	})

	t.Run("disabling a tenant shuts out tokens already issued", func(t *testing.T) {
		_, err := admin.SetStatus(ta.ID, models.TenantStatusDisabled)
		require.NoError(t, err)
		code, msg := tenantGET(t, asA(nil), "/api/admin/info")
		assert.Equal(t, 403, code)
		assert.Contains(t, msg, apperrors.ErrTenantDisabled.Code)

		_, err = admin.SetStatus(ta.ID, models.TenantStatusActive)
		require.NoError(t, err)
		code, _ = tenantGET(t, asA(nil), "/api/admin/info")
		assert.Equal(t, 200, code)
	})

	t.Run("maintenance shuts out tokens already issued", func(t *testing.T) {
		_, err := admin.SetMaintenance(ta.ID, true, "")
		require.NoError(t, err)
		code, msg := tenantGET(t, asA(nil), "/api/admin/info")
		assert.Equal(t, 503, code)
		assert.Contains(t, msg, apperrors.ErrTenantMaintenance.Code)

		_, err = admin.SetMaintenance(ta.ID, false, "")
		require.NoError(t, err)
		code, _ = tenantGET(t, asA(nil), "/api/admin/info")
		assert.Equal(t, 200, code)
	})

	t.Run("a reused tenant code inherits nothing from the old tenant", func(t *testing.T) {
		reused := "secr" + suffix
		old := create(reused)
		oldToken := tenantAdminLogin(t, reused)
		marker := "OLDTENANT" + suffix
		_, err := appfacades.OrmQuery(tenantCtx(old)).Model(&models.Admin{}).Where("id", 1).Update("nickname", marker)
		require.NoError(t, err)

		// The usual platform path: delete (database kept), then delete permanently later.
		_, err = admin.DeleteTenant(old.ID, reused, services.TenantDeleteOptions{})
		require.NoError(t, err)
		require.NoError(t, admin.ForceDeleteTenant(old.ID, reused))
		exists, err := tenantDatabaseExists(old.Database)
		require.NoError(t, err)
		assert.False(t, exists, "permanent delete must drop the database %s", old.Database)

		fresh := create(reused) // same code, same database name
		code, _ := tenantGET(t, map[string]string{"X-Tenant-ID": reused, "Authorization": "Bearer " + oldToken}, "/api/admin/info")
		assert.Equal(t, 401, code, "the old tenant's token must die with it")
		var nick []string
		require.NoError(t, appfacades.OrmQuery(tenantCtx(fresh)).Model(&models.Admin{}).Pluck("nickname", &nick))
		assert.NotContains(t, nick, marker, "the new tenant must not see the old tenant's data")
	})

	t.Run("an existing database is never adopted", func(t *testing.T) {
		code := "secx" + suffix
		dbName := services.DefaultTenantDatabaseName(code)
		q := appfacades.PlatformQuery()
		createSQL := fmt.Sprintf("CREATE DATABASE `%s`", dbName)
		dropSQL := fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", dbName)
		if isPostgres(q) {
			createSQL = fmt.Sprintf(`CREATE DATABASE "%s"`, dbName)
			dropSQL = fmt.Sprintf(`DROP DATABASE IF EXISTS "%s" WITH (FORCE)`, dbName)
		}
		_, err := q.Exec(createSQL)
		require.NoError(t, err)
		t.Cleanup(func() { _, _ = appfacades.PlatformQuery().Exec(dropSQL) })

		_, err = admin.Create(services.TenantCreateInput{Code: code, Name: code})
		require.Error(t, err)
		be, ok := apperrors.GetBusinessError(err)
		require.True(t, ok, "%v", err)
		assert.Equal(t, apperrors.ErrTenantStorageExists.Code, be.Code)
		exists, err := tenantDatabaseExists(dbName)
		require.NoError(t, err)
		assert.True(t, exists, "a database that is not the tenant's must be left alone")

		// The failed attempt leaves nothing behind: no row in the recycle bin holding the code.
		n, err := appfacades.PlatformQuery().Model(&models.Tenant{}).WithTrashed().Where("code", code).Count()
		require.NoError(t, err)
		assert.Zero(t, n, "a failed create must not keep the code")

		_, err = appfacades.PlatformQuery().Exec(dropSQL)
		require.NoError(t, err)
		tn := create(code) // the same code works right away
		assert.Equal(t, code, tn.Code)
	})

	t.Run("unknown tenant is not found, not the platform", func(t *testing.T) {
		code, msg := tenantGET(t, map[string]string{"X-Tenant-ID": "nosuch" + suffix, "Authorization": "Bearer " + tokenA}, "/api/admin/info")
		assert.Equal(t, 404, code)
		assert.True(t, strings.Contains(msg, apperrors.ErrTenantNotFound.Code), msg)
	})
}

// The platform console's permanent delete (DELETE /tenants/{id}/force, the queued purge
// path by default) drops the tenant's database before it frees the code.
func TestPlatformForceDeleteDropsTenantDatabase(t *testing.T) {
	withTenancyDriver(t, "database")
	allowSharedTenantDBCredentials(t)
	runQueueInline(t)
	platformToken := loginPlatformSmoke(t)

	admin := services.NewTenantAdminService()
	conn := services.NewTenantConnectionService()
	code := fmt.Sprintf("forcedel%d", time.Now().UnixNano()%1000000)
	tn, err := admin.Create(services.TenantCreateInput{Code: code, Name: code})
	if err != nil {
		skipWithoutTenantDB(t, err)
	}
	t.Cleanup(func() {
		_ = conn.DropStorage(tn)
		_, _ = appfacades.PlatformQuery().WithTrashed().Where("id", tn.ID).ForceDelete(&models.Tenant{})
	})
	require.NoError(t, conn.MigrateTenant(tn))

	_, err = admin.DeleteTenant(tn.ID, code, services.TenantDeleteOptions{}) // database kept
	require.NoError(t, err)
	resp, body := platformCall(t, platformToken, "DELETE", fmt.Sprintf("/api/platform/tenants/%d/force", tn.ID), fmt.Sprintf(`{"confirm_code":%q}`, code))
	require.True(t, resp.IsSuccessful(), body)

	n, err := appfacades.PlatformQuery().Model(&models.Tenant{}).WithTrashed().Where("id", tn.ID).Count()
	require.NoError(t, err)
	assert.Zero(t, n, "the tenant row is gone (code freed)")
	exists, err := tenantDatabaseExists(tn.Database)
	require.NoError(t, err)
	assert.False(t, exists, "the tenant database must be dropped before the code is freed")
}
