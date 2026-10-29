package feature_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	testinghttp "github.com/goravel/framework/contracts/testing/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/tests"
)

// platformCall sends an authenticated JSON request to the platform API.
func platformCall(t *testing.T, token, method, uri, body string) (testinghttp.Response, string) {
	t.Helper()
	testCase := tests.TestCase{}
	req := testCase.Http(t).
		WithHeader("Content-Type", "application/json").
		WithHeader("Authorization", "Bearer "+token)

	var (
		resp testinghttp.Response
		err  error
	)
	switch method {
	case "GET":
		resp, err = req.Get(uri)
	case "POST":
		resp, err = req.Post(uri, strings.NewReader(body))
	case "PUT":
		resp, err = req.Put(uri, strings.NewReader(body))
	case "DELETE":
		resp, err = req.Delete(uri, strings.NewReader(body))
	default:
		t.Fatalf("unsupported method %s", method)
	}
	require.NoError(t, err)
	content, err := resp.Content()
	require.NoError(t, err)
	return resp, content
}

func createPlatformViewer(t *testing.T) (string, string) {
	t.Helper()
	username := fmt.Sprintf("smoke_platform_ro_%d", time.Now().UnixNano()%1000000)
	password := "SmokeReadonly123!"
	_, err := services.UpsertPlatformAdmin(username, password, "Readonly", "viewer")
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = appfacades.PlatformOrmQuery(nil).Where("username", username).ForceDelete(&models.PlatformAdmin{})
	})
	return username, password
}

func TestPlatformRequiresAuth(t *testing.T) {
	withTenancyDriver(t, "database")

	testCase := tests.TestCase{}
	for _, token := range []string{"", "not-a-real-token"} {
		resp, err := testCase.Http(t).
			WithHeader("Authorization", "Bearer "+token).
			Get("/api/platform/tenants")
		require.NoError(t, err)
		resp.AssertUnauthorized()
	}
}

// A viewer may read the console but every mutation must be rejected by PlatformOwner.
// Targets are throwaway fixtures, so a missing guard cannot touch real data.
func TestPlatformViewerCannotMutate(t *testing.T) {
	withTenancyDriver(t, "database")
	viewerUser, viewerPass := createPlatformViewer(t)
	token := loginPlatformAs(t, viewerUser, viewerPass)

	nonce := time.Now().UnixNano() % 1000000
	tenant := insertPendingTenant(t, fmt.Sprintf("ro%d", nonce))
	cleanupTenantEntitlementRows(t, tenant.ID)
	planCode := fmt.Sprintf("zz_ro_plan_%d", nonce)
	newAdmin := fmt.Sprintf("smoke_ro_target_%d", nonce)
	t.Cleanup(func() {
		_, _ = appfacades.PlatformOrmQuery(nil).Where("code", planCode).ForceDelete(&models.PlatformPlan{})
		_, _ = appfacades.PlatformOrmQuery(nil).Where("username", newAdmin).ForceDelete(&models.PlatformAdmin{})
	})

	for _, uri := range []string{"/api/platform/tenants", "/api/platform/plans", fmt.Sprintf("/api/platform/tenants/%d", tenant.ID)} {
		resp, content := platformCall(t, token, "GET", uri, "")
		assert.True(t, resp.IsSuccessful(), "viewer should read %s: %s", uri, content)
	}

	tenantURI := fmt.Sprintf("/api/platform/tenants/%d", tenant.ID)
	mutations := []struct {
		method, uri, body string
	}{
		{"PUT", "/api/platform/settings", `{}`},
		{"POST", "/api/platform/admins", fmt.Sprintf(`{"username":%q,"password":"SmokeTarget123!","name":"x","role":"owner"}`, newAdmin)},
		{"POST", "/api/platform/plans", fmt.Sprintf(`{"code":%q,"name":"x"}`, planCode)},
		{"PUT", tenantURI + "/status", `{"status":0}`},
		{"PUT", tenantURI + "/maintenance", `{"enabled":true}`},
		{"POST", tenantURI + "/subscription", `{"plan_code":"free"}`},
		{"DELETE", tenantURI, ``},
	}
	for _, m := range mutations {
		t.Run(m.method+" "+m.uri, func(t *testing.T) {
			resp, content := platformCall(t, token, m.method, m.uri, m.body)
			resp.AssertForbidden()
			assert.Contains(t, content, "platform_readonly")
		})
	}

	// The fixtures are untouched.
	var after models.Tenant
	require.NoError(t, appfacades.PlatformOrmQuery(nil).Where("id", tenant.ID).First(&after))
	assert.Equal(t, tenant.Status, after.Status)
	exists, err := appfacades.PlatformOrmQuery(nil).Model(&models.PlatformPlan{}).Where("code", planCode).Exists()
	require.NoError(t, err)
	assert.False(t, exists)
}
