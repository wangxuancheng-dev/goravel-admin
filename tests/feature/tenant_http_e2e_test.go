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

// runQueueInline makes this test's queue dispatches run synchronously (the platform
// migrate / seed endpoints enqueue a TenantOps job), so nothing is left on a real queue.
func runQueueInline(t *testing.T) {
	t.Helper()
	conn := facades.Config().GetString("queue.default")
	key := "queue.connections." + conn + ".driver"
	prev := facades.Config().GetString(key)
	facades.Config().Add(key, "sync")
	t.Cleanup(func() { facades.Config().Add(key, prev) })
}

// tenantAdminCall calls the tenant admin API as tenantCode with an admin token.
func tenantAdminCall(t *testing.T, tenantCode, token, method, uri, body string) (int, map[string]any) {
	t.Helper()
	tc := &tests.TestCase{}
	req := tc.Http(t).WithHeader("X-Tenant-ID", tenantCode).WithHeader("Content-Type", "application/json")
	if token != "" {
		req = req.WithHeader("Authorization", "Bearer "+token)
	}
	var (
		content string
		err     error
	)
	switch method {
	case "GET":
		resp, e := req.Get(uri)
		require.NoError(t, e)
		content, err = resp.Content()
	case "POST":
		resp, e := req.Post(uri, strings.NewReader(body))
		require.NoError(t, e)
		content, err = resp.Content()
	case "PUT":
		resp, e := req.Put(uri, strings.NewReader(body))
		require.NoError(t, e)
		content, err = resp.Content()
	}
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal([]byte(content), &payload), "body: %s", content)
	code, _ := payload["code"].(float64)
	return int(code), payload
}

func tenantAdminLogin(t *testing.T, tenantCode string) string {
	t.Helper()
	code, payload := tenantAdminCall(t, tenantCode, "", "POST", "/api/admin/login", `{"username":"admin","password":"admin123"}`)
	require.Equal(t, 200, code, "login as tenant %s: %v", tenantCode, payload)
	data, _ := payload["data"].(map[string]any)
	token, _ := data["token"].(string)
	require.NotEmpty(t, token)
	return token
}

func adminNickname(payload map[string]any) string {
	data, _ := payload["data"].(map[string]any)
	if admin, ok := data["admin"].(map[string]any); ok {
		if n, ok := admin["nickname"].(string); ok {
			return n
		}
	}
	if n, ok := data["nickname"].(string); ok {
		return n
	}
	return ""
}

// End to end over HTTP: the platform migrates and seeds tenants through its API and
// the queue job, then each tenant's admin only ever reaches its own database.
func TestTenantLifecycleAndIsolationOverHTTP(t *testing.T) {
	withTenancyDriver(t, "database")
	allowSharedTenantDBCredentials(t)
	require.True(t, tenancy.Enabled())
	runQueueInline(t)
	platformToken := loginPlatformSmoke(t)

	admin := services.NewTenantAdminService()
	conn := services.NewTenantConnectionService()
	suffix := fmt.Sprint(time.Now().UnixNano() % 1000000)
	create := func(prefix string) *models.Tenant {
		tn, err := admin.Create(services.TenantCreateInput{Code: prefix + suffix, Name: prefix, Migrate: false})
		if err != nil {
			skipWithoutTenantDB(t, err)
		}
		t.Cleanup(func() {
			_ = conn.DropStorage(tn)
			_, _ = appfacades.PlatformQuery().WithTrashed().Where("id", tn.ID).ForceDelete(&models.Tenant{})
		})
		return tn
	}
	ta, tb := create("httpa"), create("httpb")

	// Not provisioned yet: the tenant API refuses it.
	code, _ := tenantAdminCall(t, ta.Code, "", "POST", "/api/admin/login", `{"username":"admin","password":"admin123"}`)
	assert.NotEqual(t, 200, code, "a pending tenant must not accept logins")

	// Platform: A = migrate with seed in one op; B = migrate, then a separate seed op.
	resp, content := platformCall(t, platformToken, "POST", fmt.Sprintf("/api/platform/tenants/%d/migrate", ta.ID), `{"with_seed":true}`)
	require.True(t, resp.IsSuccessful(), content)
	resp, content = platformCall(t, platformToken, "POST", fmt.Sprintf("/api/platform/tenants/%d/migrate", tb.ID), `{}`)
	require.True(t, resp.IsSuccessful(), content)
	resp, content = platformCall(t, platformToken, "POST", fmt.Sprintf("/api/platform/tenants/%d/seed", tb.ID), `{}`)
	require.True(t, resp.IsSuccessful(), content)

	for _, tn := range []*models.Tenant{ta, tb} {
		var row models.Tenant
		require.NoError(t, appfacades.PlatformQuery().Where("id", tn.ID).First(&row))
		assert.Equal(t, models.TenantProvisionReady, row.ProvisionStatus, "tenant %s provision", tn.Code)
		assert.Equal(t, models.TenantOpStatusSuccess, row.LastOpStatus, "tenant %s last op: %s", tn.Code, row.LastOpMessage)
		assert.Positive(t, row.SchemaMigrationCount)
		// Tenant tables live in the tenant database, not in the platform one.
		n, err := appfacades.OrmQuery(tenantCtx(tn)).Table("admins").Count()
		require.NoError(t, err)
		assert.Positive(t, n, "tenant %s must be seeded", tn.Code)
	}

	tokenA := tenantAdminLogin(t, ta.Code)
	tokenB := tenantAdminLogin(t, tb.Code)

	code, infoA := tenantAdminCall(t, ta.Code, tokenA, "GET", "/api/admin/info", "")
	require.Equal(t, 200, code, "%v", infoA)
	assert.Contains(t, fmt.Sprint(infoA), ta.Code, "info must report the bound tenant")

	// A token is only valid in the tenant database it was issued by.
	code, _ = tenantAdminCall(t, tb.Code, tokenA, "GET", "/api/admin/info", "")
	assert.Equal(t, 401, code, "tenant A's token must be rejected for tenant B")
	code, _ = tenantAdminCall(t, ta.Code, tokenB, "GET", "/api/admin/info", "")
	assert.Equal(t, 401, code, "tenant B's token must be rejected for tenant A")

	// Writes through tenant A's API land in A's database only.
	nick := "only-in-a-" + suffix
	code, upd := tenantAdminCall(t, ta.Code, tokenA, "PUT", "/api/admin/profile", fmt.Sprintf(`{"nickname":%q}`, nick))
	require.Equal(t, 200, code, "%v", upd)
	_, infoA = tenantAdminCall(t, ta.Code, tokenA, "GET", "/api/admin/info", "")
	_, infoB := tenantAdminCall(t, tb.Code, tokenB, "GET", "/api/admin/info", "")
	assert.Equal(t, nick, adminNickname(infoA))
	assert.NotEqual(t, nick, adminNickname(infoB), "tenant B must not see tenant A's write")

	platformAdmins, err := appfacades.PlatformQuery().Table("admins").Where("nickname", nick).Count()
	require.NoError(t, err)
	assert.Zero(t, platformAdmins, "tenant writes must not reach the platform database")
}
