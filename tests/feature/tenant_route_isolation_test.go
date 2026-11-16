package feature_test

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/goravel/framework/facades"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/tests"
)

// Routes the scan does not call: streams that never end, redirects to an IdP, file
// bodies, and the host monitor (slow, and host data rather than tenant data).
var routeScanSkip = []string{
	"/stream", "/oidc", "/download", "/preview", "/error-file", "/monitor/",
	"/public/images/", "/observability/pprof",
}

// tenantAdminGETRoutes lists every GET route of the tenant admin API (resource
// routes by their index), with {id} filled with 1 and other parameters skipped. New
// routes are scanned without anyone listing them.
func tenantAdminGETRoutes() []string {
	seen := map[string]bool{}
	for _, r := range facades.Route().GetRoutes() {
		if !strings.HasPrefix(r.Path, "/api/admin/") {
			continue
		}
		if !strings.Contains(r.Method, "GET") && r.Method != "RESOURCE" {
			continue
		}
		path := strings.ReplaceAll(r.Path, "{id}", "1")
		if strings.Contains(path, "{") {
			continue
		}
		skip := false
		for _, s := range routeScanSkip {
			if strings.Contains(path, s) {
				skip = true
				break
			}
		}
		if !skip {
			seen[path] = true
			if r.Method == "RESOURCE" {
				seen[path+"/1"] = true // show
			}
		}
	}
	out := make([]string, 0, len(seen))
	for p := range seen {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

func rawTenantGET(t *testing.T, tenantCode, token, uri string, headers map[string]string) (bool, string) {
	t.Helper()
	req := (&tests.TestCase{}).Http(t).WithHeader("X-Tenant-ID", tenantCode).WithHeader("Authorization", "Bearer "+token)
	for k, v := range headers {
		req = req.WithHeader(k, v)
	}
	resp, err := req.Get(uri)
	require.NoError(t, err)
	body, err := resp.Content()
	if err != nil && err != io.EOF {
		t.Fatalf("GET %s: %v", uri, err)
	}
	return resp.IsSuccessful(), body
}

// Tenant A's data is marked with a unique string in several tables (and its login
// log); then every GET route of the tenant admin API is called as tenant B, whose
// responses must never contain it.
func TestEveryAdminRouteIsTenantIsolated(t *testing.T) {
	withTenancyDriver(t, "database")
	allowSharedTenantDBCredentials(t)

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
		require.NoError(t, conn.MigrateTenant(tn))
		require.NoError(t, conn.SeedTenant(tn))
		return tn
	}
	ta, tb := create("scana"), create("scanb")

	marker := "XTENANTLEAK" + suffix
	ctxA := tenantCtx(ta)
	_, err := appfacades.OrmQuery(ctxA).Model(&models.Admin{}).Where("id", 1).Update("nickname", marker)
	require.NoError(t, err)
	require.NoError(t, appfacades.OrmQuery(ctxA).Create(&models.Dictionary{Type: marker, Label: marker, Value: marker, Status: 1}))
	require.NoError(t, appfacades.OrmQuery(ctxA).Create(&models.Department{Name: marker, Code: marker, Status: 1}))

	// A's login log carries the marker as its User-Agent.
	code, payload := tenantAdminCallWithUA(t, ta.Code, marker)
	require.Equal(t, 200, code, "login A: %v", payload)
	tokenA := tokenFrom(t, payload)
	tokenB := tenantAdminLogin(t, tb.Code)

	routes := tenantAdminGETRoutes()
	require.Greater(t, len(routes), 30, "route scan found too few routes")

	// The marker is reachable as A (so its absence for B means something).
	_, body := rawTenantGET(t, ta.Code, tokenA, "/api/admin/info", nil)
	require.Contains(t, body, marker, "tenant A must see its own marker")

	ok := 0
	for _, uri := range routes {
		success, body := rawTenantGET(t, tb.Code, tokenB, uri, nil)
		if success {
			ok++
		}
		assert.NotContains(t, body, marker, "GET %s as tenant B returned tenant A's data", uri)
	}
	// Most routes must actually answer, or the scan proves nothing.
	assert.Greater(t, ok, len(routes)/2, "only %d of %d routes answered 2xx as tenant B", ok, len(routes))
}

func tenantAdminCallWithUA(t *testing.T, tenantCode, ua string) (int, map[string]any) {
	t.Helper()
	req := (&tests.TestCase{}).Http(t).
		WithHeader("X-Tenant-ID", tenantCode).
		WithHeader("Content-Type", "application/json").
		WithHeader("User-Agent", ua)
	resp, err := req.Post("/api/admin/login", strings.NewReader(`{"username":"admin","password":"admin123"}`))
	require.NoError(t, err)
	payload, err := resp.Json()
	require.NoError(t, err)
	code, _ := payload["code"].(float64)
	return int(code), payload
}

func tokenFrom(t *testing.T, payload map[string]any) string {
	t.Helper()
	data, _ := payload["data"].(map[string]any)
	token, _ := data["token"].(string)
	require.NotEmpty(t, token)
	return token
}
