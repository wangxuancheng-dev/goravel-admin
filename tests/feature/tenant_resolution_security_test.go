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
		code, msg := tenantGET(t, asA(map[string]string{"X-Forwarded-Host": "example.test"}), "/api/admin/info")
		assert.Equal(t, 400, code)
		assert.Contains(t, msg, apperrors.ErrTenantRequired.Code)
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

	t.Run("a reused tenant code does not honour the old tenant's tokens", func(t *testing.T) {
		reused := "secr" + suffix
		old := create(reused)
		oldToken := tenantAdminLogin(t, reused)

		require.NoError(t, conn.DropStorage(old))
		_, err := appfacades.PlatformQuery().Where("id", old.ID).ForceDelete(&models.Tenant{})
		require.NoError(t, err)

		create(reused) // same code, new database
		code, _ := tenantGET(t, map[string]string{"X-Tenant-ID": reused, "Authorization": "Bearer " + oldToken}, "/api/admin/info")
		assert.Equal(t, 401, code, "the old tenant's token must die with it")
	})

	t.Run("unknown tenant is not found, not the platform", func(t *testing.T) {
		code, msg := tenantGET(t, map[string]string{"X-Tenant-ID": "nosuch" + suffix, "Authorization": "Bearer " + tokenA}, "/api/admin/info")
		assert.Equal(t, 404, code)
		assert.True(t, strings.Contains(msg, apperrors.ErrTenantNotFound.Code), msg)
	})
}
