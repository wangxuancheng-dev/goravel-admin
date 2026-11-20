package feature_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/goravel/framework/facades"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/app/tenancy"
	"goravel/tests"
)

// Platform "sign in as tenant admin": ticket issue, single-use redeem bound to the
// tenant, hard-expiry session, blocked credential changes, tenant-side audit row.
func TestTenantImpersonationOverHTTP(t *testing.T) {
	withTenancyDriver(t, "database")
	allowSharedTenantDBCredentials(t)
	require.True(t, tenancy.Enabled())
	runQueueInline(t)
	platformToken := loginPlatformSmoke(t)

	tenantSvc := services.NewTenantAdminService()
	conn := services.NewTenantConnectionService()
	suffix := fmt.Sprint(time.Now().UnixNano() % 1000000)
	create := func(prefix string) *models.Tenant {
		tn, err := tenantSvc.Create(services.TenantCreateInput{Code: prefix + suffix, Name: prefix, Migrate: false})
		if err != nil {
			skipWithoutTenantDB(t, err)
		}
		t.Cleanup(func() {
			_ = conn.DropStorage(tn)
			_, _ = appfacades.PlatformQuery().WithTrashed().Where("id", tn.ID).ForceDelete(&models.Tenant{})
		})
		resp, content := platformCall(t, platformToken, "POST", fmt.Sprintf("/api/platform/tenants/%d/migrate", tn.ID), `{"with_seed":true}`)
		require.True(t, resp.IsSuccessful(), content)
		return tn
	}
	ta, tb := create("impa"), create("impb")

	var target models.Admin
	require.NoError(t, appfacades.OrmQuery(tenantCtx(ta)).Where("username", "admin").First(&target))
	require.NotZero(t, target.ID)

	issue := func(body string) (bool, map[string]any) {
		resp, content := platformCall(t, platformToken, "POST", fmt.Sprintf("/api/platform/tenants/%d/impersonate", ta.ID), body)
		var payload map[string]any
		require.NoError(t, json.Unmarshal([]byte(content), &payload), content)
		data, _ := payload["data"].(map[string]any)
		return resp.IsSuccessful(), data
	}

	// A reason is mandatory.
	ok, _ := issue(fmt.Sprintf(`{"admin_id":%d}`, target.ID))
	assert.False(t, ok, "impersonation without a reason must be refused")

	// A ticket for tenant A cannot be redeemed on tenant B.
	ok, data := issue(fmt.Sprintf(`{"admin_id":%d,"reason":"ticket #1 wrong tenant"}`, target.ID))
	require.True(t, ok)
	wrongTicket, _ := data["ticket"].(string)
	code, _ := tenantAdminCall(t, tb.Code, "", "POST", "/api/admin/auth/impersonate", fmt.Sprintf(`{"ticket":%q}`, wrongTicket))
	assert.NotEqual(t, 200, code, "a ticket must only work on its own tenant")

	// Happy path.
	ok, data = issue(fmt.Sprintf(`{"admin_id":%d,"reason":"ticket #2 investigate orders"}`, target.ID))
	require.True(t, ok)
	ticket, _ := data["ticket"].(string)
	require.NotEmpty(t, ticket)
	url, _ := data["url"].(string)
	assert.Contains(t, url, "/login/impersonate?")
	assert.Contains(t, url, "tenant_code="+ta.Code)
	assert.Empty(t, data["host"], "header mode opens on the console origin")
	path, _ := data["path"].(string)
	assert.True(t, strings.HasPrefix(path, "/login/impersonate?"), path)
	assert.Contains(t, path, "ticket="+ticket)

	code, payload := tenantAdminCall(t, ta.Code, "", "POST", "/api/admin/auth/impersonate", fmt.Sprintf(`{"ticket":%q}`, ticket))
	require.Equal(t, 200, code, "redeem: %v", payload)
	redeemed, _ := payload["data"].(map[string]any)
	token, _ := redeemed["token"].(string)
	require.NotEmpty(t, token)

	// Single use.
	code, _ = tenantAdminCall(t, ta.Code, "", "POST", "/api/admin/auth/impersonate", fmt.Sprintf(`{"ticket":%q}`, ticket))
	assert.NotEqual(t, 200, code, "a ticket must not be redeemable twice")

	// /info reports the impersonation.
	code, info := tenantAdminCall(t, ta.Code, token, "GET", "/api/admin/info", "")
	require.Equal(t, 200, code, "info: %v", info)
	infoData, _ := info["data"].(map[string]any)
	imp, _ := infoData["impersonation"].(map[string]any)
	require.NotNil(t, imp, "info must carry impersonation")
	assert.Equal(t, platformSmokeUser, imp["platform_username"])

	// Own credential changes are blocked.
	code, blocked := tenantAdminCall(t, ta.Code, token, "PUT", "/api/admin/password",
		`{"old_password":"admin123","new_password":"Xx9!impersonate","confirm_password":"Xx9!impersonate"}`)
	assert.Equal(t, 403, code)
	assert.Equal(t, "impersonation_forbidden", blocked["error_code"])
	code, _ = tenantAdminCall(t, ta.Code, token, "POST", "/api/admin/google-authenticator/bind", `{"secret":"x","code":"000000"}`)
	assert.Equal(t, 403, code)
	code, blocked = tenantAdminCall(t, ta.Code, token, "DELETE", "/api/admin/auth/tokens/others", "")
	assert.Equal(t, 403, code, "support must not sign the real admin out: %v", blocked)

	// Writes made in the session are tagged with the platform account in the tenant
	// operation log (written asynchronously), and the list can filter on it.
	nick := "support-edit-" + suffix
	code, upd := tenantAdminCall(t, ta.Code, token, "PUT", "/api/admin/profile", fmt.Sprintf(`{"nickname":%q}`, nick))
	require.Equal(t, 200, code, "profile update while impersonating: %v", upd)
	var opRow models.OperationLog
	require.Eventually(t, func() bool {
		opRow = models.OperationLog{}
		err := appfacades.OrmQuery(tenantCtx(ta)).Where("admin_id", target.ID).
			Where("path", "/api/admin/profile").Where("impersonator <> ''").
			Order("id desc").First(&opRow)
		return err == nil && opRow.ID != 0
	}, 10*time.Second, 200*time.Millisecond, "impersonated write must be in the operation log")
	assert.Equal(t, fmt.Sprintf("%s#%v", platformSmokeUser, imp["platform_admin_id"]), opRow.Impersonator)
	code, list := tenantAdminCall(t, ta.Code, token, "GET", "/api/admin/operation-logs?impersonated=1&start_time=&page_size=50", "")
	require.Equal(t, 200, code, "operation log list: %v", list)
	assert.Contains(t, fmt.Sprint(list), platformSmokeUser+"#", "impersonated=1 must list the tagged row")
	code, list = tenantAdminCall(t, ta.Code, token, "GET", "/api/admin/operation-logs?impersonated=0&start_time=&page_size=50", "")
	require.Equal(t, 200, code, "operation log list: %v", list)
	assert.NotContains(t, fmt.Sprint(list), platformSmokeUser+"#", "impersonated=0 must hide the tagged row")

	// A tenant not yet migrated to the impersonator column still logs normal writes.
	_, err := appfacades.OrmQuery(tenantCtx(tb)).Exec("ALTER TABLE operation_logs DROP COLUMN impersonator")
	require.NoError(t, err)
	tokenB := tenantAdminLogin(t, tb.Code)
	nickB := "pre-migrate-" + suffix
	code, upd = tenantAdminCall(t, tb.Code, tokenB, "PUT", "/api/admin/profile", fmt.Sprintf(`{"nickname":%q}`, nickB))
	require.Equal(t, 200, code, "profile update: %v", upd)
	require.Eventually(t, func() bool {
		n, countErr := appfacades.OrmQuery(tenantCtx(tb)).Model(&models.OperationLog{}).
			Where("path", "/api/admin/profile").Count()
		return countErr == nil && n > 0
	}, 10*time.Second, 200*time.Millisecond, "operation log must be written without the impersonator column")

	// A tenant in maintenance rejects every admin request: refuse the ticket up front.
	_, err = appfacades.PlatformQuery().Model(&models.Tenant{}).Where("id", tb.ID).Update("maintenance", true)
	require.NoError(t, err)
	resp, content := platformCall(t, platformToken, "POST", fmt.Sprintf("/api/platform/tenants/%d/impersonate", tb.ID),
		`{"admin_id":1,"reason":"maintenance check"}`)
	assert.False(t, resp.IsSuccessful(), content)
	assert.Contains(t, content, "tenant_maintenance")

	// Hard expiry: the stored token is tagged and expires at issue + TTL, never extended.
	var pat models.PersonalAccessToken
	require.NoError(t, appfacades.OrmQuery(tenantCtx(ta)).
		Where("tokenable_id", target.ID).
		Where("name LIKE ?", services.ImpersonationTokenPrefix+"%").
		Order("id desc").First(&pat))
	require.NotNil(t, pat.ExpiresAt)
	assert.True(t, strings.HasSuffix(pat.Name, ":"+platformSmokeUser), pat.Name)
	assert.WithinDuration(t, time.Now().Add(services.ImpersonationTTL()), *pat.ExpiresAt, 2*time.Minute)

	// Tenant-side audit: login log row with the platform user and reason.
	var logRow models.LoginLog
	require.NoError(t, appfacades.OrmQuery(tenantCtx(ta)).
		Where("admin_id", target.ID).Where("message", "impersonate_login").
		Order("id desc").First(&logRow))
	require.NotZero(t, logRow.ID, "tenant login log must record the impersonation")
	assert.Contains(t, logRow.Request, "ticket #2 investigate orders")
	assert.Contains(t, logRow.Request, platformSmokeUser)

	// Subdomain resolver: the link points at the tenant's own host, and the landing page
	// there (Origin = tenant subdomain, plus the tenant_code it stores) redeems it.
	setTenancyConfig(t, "tenancy.resolver", "subdomain")
	setTenancyConfig(t, "tenancy.base_domain", "imp.example.test")
	ok, data = issue(fmt.Sprintf(`{"admin_id":%d,"reason":"subdomain mode"}`, target.ID))
	require.True(t, ok)
	subHost := ta.Code + ".imp.example.test"
	assert.Equal(t, subHost, data["host"])
	ticket, _ = data["ticket"].(string)
	origin := "http://" + subHost + ":3008"

	// Another tenant's subdomain cannot redeem it.
	code, _ = impersonateRedeem(t, "http://"+tb.Code+".imp.example.test:3008", tb.Code, ticket)
	assert.NotEqual(t, 200, code)
	ok, data = issue(fmt.Sprintf(`{"admin_id":%d,"reason":"subdomain mode 2"}`, target.ID))
	require.True(t, ok)
	ticket, _ = data["ticket"].(string)
	code, payload = impersonateRedeem(t, origin, ta.Code, ticket)
	require.Equal(t, 200, code, "redeem on the tenant subdomain: %v", payload)
	// Without a stored tenant_code (Host/Origin only) it works too.
	ok, data = issue(fmt.Sprintf(`{"admin_id":%d,"reason":"subdomain mode 3"}`, target.ID))
	require.True(t, ok)
	ticket, _ = data["ticket"].(string)
	code, payload = impersonateRedeem(t, origin, "", ticket)
	require.Equal(t, 200, code, "redeem by Origin only: %v", payload)
}

func setTenancyConfig(t *testing.T, key, value string) {
	t.Helper()
	prev := facades.Config().GetString(key)
	facades.Config().Add(key, value)
	t.Cleanup(func() { facades.Config().Add(key, prev) })
}

// impersonateRedeem posts the ticket like the landing page on a tenant host would.
func impersonateRedeem(t *testing.T, origin, tenantCode, ticket string) (int, map[string]any) {
	t.Helper()
	req := (&tests.TestCase{}).Http(t).WithHeader("Content-Type", "application/json").WithHeader("Origin", origin)
	if tenantCode != "" {
		req = req.WithHeader("X-Tenant-ID", tenantCode)
	}
	resp, err := req.Post("/api/admin/auth/impersonate", strings.NewReader(fmt.Sprintf(`{"ticket":%q}`, ticket)))
	require.NoError(t, err)
	content, err := resp.Content()
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal([]byte(content), &payload), content)
	c, _ := payload["code"].(float64)
	return int(c), payload
}
