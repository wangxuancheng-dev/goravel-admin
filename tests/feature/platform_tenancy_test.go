package feature_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/goravel/framework/facades"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appfacades "goravel/app/facades"
	"goravel/app/search"
	"goravel/app/services"
	"goravel/app/tenancy"
	"goravel/app/tenancyctx"
	"goravel/tests"
)

const (
	platformSmokeUser = "smoke_platform_admin"
	platformSmokePass = "SmokePlatform123!"
)

func withTenancyDriver(t *testing.T, driver string) {
	t.Helper()
	prev := facades.Config().GetString("tenancy.driver", "off")
	facades.Config().Add("tenancy.driver", driver)
	t.Cleanup(func() {
		facades.Config().Add("tenancy.driver", prev)
	})
}

func ensurePlatformSmokeAdmin(t *testing.T) {
	t.Helper()
	_, err := services.UpsertPlatformAdmin(platformSmokeUser, platformSmokePass, "Smoke Platform", "owner")
	require.NoError(t, err)
}

func loginPlatformSmoke(t *testing.T) string {
	t.Helper()
	ensurePlatformSmokeAdmin(t)

	body := fmt.Sprintf(`{"username":%q,"password":%q}`, platformSmokeUser, platformSmokePass)
	testCase := tests.TestCase{}
	resp, err := testCase.Http(t).
		WithHeader("Content-Type", "application/json").
		Post("/api/platform/login", strings.NewReader(body))
	require.NoError(t, err)
	resp.AssertOk()

	content, err := resp.Content()
	require.NoError(t, err)

	var payload struct {
		Code int `json:"code"`
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(content), &payload))
	require.Equal(t, 200, payload.Code)
	require.NotEmpty(t, payload.Data.Token)
	return payload.Data.Token
}

func TestPlatformLoginRequiresTenancy(t *testing.T) {
	withTenancyDriver(t, "off")

	body := `{"username":"x","password":"y"}`
	testCase := tests.TestCase{}
	resp, err := testCase.Http(t).
		WithHeader("Content-Type", "application/json").
		Post("/api/platform/login", strings.NewReader(body))
	require.NoError(t, err)

	content, err := resp.Content()
	require.NoError(t, err)
	assert.Contains(t, content, "tenancy_disabled")
}

func TestPlatformLoginInfoAndTenants(t *testing.T) {
	withTenancyDriver(t, "database")
	token := loginPlatformSmoke(t)

	testCase := tests.TestCase{}
	infoResp, err := testCase.Http(t).
		WithHeader("Authorization", "Bearer "+token).
		Get("/api/platform/info")
	require.NoError(t, err)
	infoResp.AssertOk()

	infoBody, err := infoResp.Content()
	require.NoError(t, err)
	assert.Contains(t, infoBody, platformSmokeUser)

	listResp, err := testCase.Http(t).
		WithHeader("Authorization", "Bearer "+token).
		Get("/api/platform/tenants")
	require.NoError(t, err)
	listResp.AssertOk()

	healthResp, err := testCase.Http(t).
		WithHeader("Authorization", "Bearer "+token).
		Get("/api/platform/health")
	require.NoError(t, err)
	healthResp.AssertOk()
	healthBody, err := healthResp.Content()
	require.NoError(t, err)
	assert.Contains(t, healthBody, `"driver"`)
	assert.Contains(t, healthBody, "database")
}

func TestAdminRequiresTenantWhenTenancyOn(t *testing.T) {
	withTenancyDriver(t, "database")

	testCase := tests.TestCase{}
	resp, err := testCase.Http(t).
		WithHeader("Authorization", "Bearer unused").
		Get("/api/admin/info")
	require.NoError(t, err)
	// Tenant middleware runs before JWT — missing hint → tenant_required
	content, err := resp.Content()
	require.NoError(t, err)
	assert.Contains(t, content, "tenant_required")
}

func TestAdminUnknownTenantRejected(t *testing.T) {
	withTenancyDriver(t, "database")

	testCase := tests.TestCase{}
	resp, err := testCase.Http(t).
		WithHeader("Authorization", "Bearer unused").
		WithHeader("X-Tenant-ID", "no_such_tenant_code_xyz").
		Get("/api/admin/info")
	require.NoError(t, err)
	content, err := resp.Content()
	require.NoError(t, err)
	assert.Contains(t, content, "tenant_not_found")
}

func TestPlatformChangePassword(t *testing.T) {
	withTenancyDriver(t, "database")
	token := loginPlatformSmoke(t)

	body := fmt.Sprintf(
		`{"old_password":%q,"new_password":%q,"confirm_password":%q}`,
		platformSmokePass,
		platformSmokePass+"X",
		platformSmokePass+"X",
	)
	testCase := tests.TestCase{}
	resp, err := testCase.Http(t).
		WithHeader("Authorization", "Bearer "+token).
		WithHeader("Content-Type", "application/json").
		Put("/api/platform/password", strings.NewReader(body))
	require.NoError(t, err)
	resp.AssertOk()

	// restore for other tests
	_, err = services.UpsertPlatformAdmin(platformSmokeUser, platformSmokePass, "Smoke Platform", "owner")
	require.NoError(t, err)
}

func TestPlatformCreateRejectsSyncMigrate(t *testing.T) {
	withTenancyDriver(t, "database")
	token := loginPlatformSmoke(t)

	body := `{"code":"sync_migrate_reject","name":"Reject Sync","migrate":true}`
	testCase := tests.TestCase{}
	resp, err := testCase.Http(t).
		WithHeader("Authorization", "Bearer "+token).
		WithHeader("Content-Type", "application/json").
		Post("/api/platform/tenants", strings.NewReader(body))
	require.NoError(t, err)
	content, err := resp.Content()
	require.NoError(t, err)
	assert.Contains(t, content, "tenant_migrate_via_cli")
}

func TestTenancyCacheAndSearchIndexIsolation(t *testing.T) {
	withTenancyDriver(t, "database")

	ctxA := tenancyctx.WithTenant(context.Background(), 1, "tenant_a", "alpha")
	ctxB := tenancyctx.WithTenant(context.Background(), 2, "tenant_b", "beta")

	assert.NotEqual(t, tenancy.CacheKey(ctxA, "lock:x"), tenancy.CacheKey(ctxB, "lock:x"))
	assert.NotEqual(t, tenancy.StoragePrefix(ctxA), tenancy.StoragePrefix(ctxB))
	assert.Equal(t, "tenants/alpha/", tenancy.StoragePrefix(ctxA))
	assert.Equal(t, "tenants/beta/", tenancy.StoragePrefix(ctxB))

	idxA := search.OrdersIndexShortNameFor(ctxA)
	idxB := search.OrdersIndexShortNameFor(ctxB)
	assert.NotEqual(t, idxA, idxB)
	assert.True(t, strings.HasPrefix(idxA, "alpha_"))
	assert.True(t, strings.HasPrefix(idxB, "beta_"))

	assert.NotEqual(t, appfacades.SchemaConnectionKeyFrom(ctxA), appfacades.SchemaConnectionKeyFrom(ctxB))
}
