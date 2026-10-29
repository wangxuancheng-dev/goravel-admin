package feature_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/goravel/framework/facades"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/app/tenancyctx"
	"goravel/tests"
)

// tenantAdminGate returns the body of a tenant-side request, which the tenant middleware
// answers before any database access for disabled / maintenance / unknown tenants.
func tenantAdminGate(t *testing.T, code string) string {
	t.Helper()
	testCase := tests.TestCase{}
	resp, err := testCase.Http(t).
		WithHeader("Authorization", "Bearer unused").
		WithHeader("X-Tenant-ID", code).
		Get("/api/admin/info")
	require.NoError(t, err)
	content, err := resp.Content()
	require.NoError(t, err)
	return content
}

func TestPlatformTenantStatusAndMaintenanceGateTenantRequests(t *testing.T) {
	withTenancyDriver(t, "database")
	token := loginPlatformSmoke(t)
	tenant := insertTenantFixture(t, fmt.Sprintf("gate%d", time.Now().UnixNano()%1000000), models.TenantProvisionReady)
	uri := fmt.Sprintf("/api/platform/tenants/%d", tenant.ID)

	resp, content := platformCall(t, token, "PUT", uri+"/status", `{"status":0}`)
	require.True(t, resp.IsSuccessful(), content)
	assert.Contains(t, tenantAdminGate(t, tenant.Code), "tenant_disabled")

	resp, content = platformCall(t, token, "PUT", uri+"/status", `{"status":1}`)
	require.True(t, resp.IsSuccessful(), content)

	resp, content = platformCall(t, token, "PUT", uri+"/maintenance", `{"maintenance":true}`)
	require.True(t, resp.IsSuccessful(), content)
	gate := tenantAdminGate(t, tenant.Code)
	assert.Contains(t, gate, "tenant_maintenance")
	var generic struct {
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal([]byte(gate), &generic))
	assert.NotEmpty(t, generic.Message, "without a notice the translated default is shown")

	// The operator's notice reaches tenant users instead of the generic text.
	resp, content = platformCall(t, token, "PUT", uri+"/maintenance", `{"maintenance":true,"maintenance_message":"upgrading storage until 22:00"}`)
	require.True(t, resp.IsSuccessful(), content)
	gate = tenantAdminGate(t, tenant.Code)
	assert.Contains(t, gate, "tenant_maintenance")
	assert.Contains(t, gate, "upgrading storage until 22:00")

	// Long notices are cut to the 500-character column without splitting a multi-byte character.
	long := strings.Repeat("系统升级", 200)
	resp, content = platformCall(t, token, "PUT", uri+"/maintenance", fmt.Sprintf(`{"maintenance":true,"maintenance_message":%q}`, long))
	require.True(t, resp.IsSuccessful(), content)
	var notice struct {
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal([]byte(tenantAdminGate(t, tenant.Code)), &notice))
	assert.Equal(t, string([]rune(long)[:500]), notice.Message)

	resp, content = platformCall(t, token, "PUT", uri+"/maintenance", `{"maintenance":false}`)
	require.True(t, resp.IsSuccessful(), content)
	gate = tenantAdminGate(t, tenant.Code)
	assert.NotContains(t, gate, "tenant_maintenance")
	assert.NotContains(t, gate, "tenant_disabled")

	var stored models.Tenant
	require.NoError(t, appfacades.PlatformOrmQuery(nil).Where("id", tenant.ID).First(&stored))
	assert.False(t, stored.Maintenance)
	assert.Empty(t, stored.MaintenanceMessage, "turning maintenance off clears the message")

	for _, body := range []string{`{"status":5}`, `{}`} {
		resp, content = platformCall(t, token, "PUT", uri+"/status", body)
		resp.AssertStatus(400)
		assert.Contains(t, content, "invalid_argument")
	}
	resp, content = platformCall(t, token, "PUT", uri+"/maintenance", `{}`)
	resp.AssertStatus(400)
	assert.Contains(t, content, "invalid_argument")
}

func TestPlatformTenantDeleteAndUndelete(t *testing.T) {
	withTenancyDriver(t, "database")
	token := loginPlatformSmoke(t)
	tenant := insertTenantFixture(t, fmt.Sprintf("del%d", time.Now().UnixNano()%1000000), models.TenantProvisionReady)
	uri := fmt.Sprintf("/api/platform/tenants/%d", tenant.ID)

	// Guards: wrong confirm code, and recycle-bin-only actions on a live tenant.
	resp, content := platformCall(t, token, "DELETE", uri, `{"confirm_code":"not-the-code"}`)
	resp.AssertStatus(400)
	assert.Contains(t, content, "invalid_argument")
	resp, content = platformCall(t, token, "POST", uri+"/undelete", ``)
	assert.False(t, resp.IsSuccessful())
	assert.Contains(t, content, "tenant_not_trashed")
	resp, content = platformCall(t, token, "DELETE", uri+"/force", fmt.Sprintf(`{"confirm_code":%q}`, tenant.Code))
	assert.False(t, resp.IsSuccessful())
	assert.Contains(t, content, "tenant_not_trashed")

	resp, content = platformCall(t, token, "DELETE", uri, fmt.Sprintf(`{"confirm_code":%q}`, tenant.Code))
	require.True(t, resp.IsSuccessful(), content)
	// The console can still open a recycle-bin tenant, flagged by deleted_at.
	assert.NotEmpty(t, tenantDeletedAt(t, token, uri))
	assert.Contains(t, tenantAdminGate(t, tenant.Code), "tenant_not_found", "a deleted tenant must not resolve")

	resp, content = platformCall(t, token, "POST", uri+"/undelete", ``)
	require.True(t, resp.IsSuccessful(), content)
	assert.Empty(t, tenantDeletedAt(t, token, uri))
	// Undelete pings the tenant database; the fixture has none, so the tenant comes back
	// resolvable but flagged not-ready instead of silently serving traffic.
	assert.Contains(t, tenantAdminGate(t, tenant.Code), "tenant_not_ready")
}

func tenantDeletedAt(t *testing.T, token, uri string) string {
	t.Helper()
	resp, content := platformCall(t, token, "GET", uri, "")
	require.True(t, resp.IsSuccessful(), content)
	var payload struct {
		Data struct {
			Tenant struct {
				DeletedAt *string `json:"deleted_at"`
			} `json:"tenant"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(content), &payload))
	if payload.Data.Tenant.DeletedAt == nil {
		return ""
	}
	return *payload.Data.Tenant.DeletedAt
}

func TestPlatformTenantBackupFileGuards(t *testing.T) {
	withTenancyDriver(t, "database")
	token := loginPlatformSmoke(t)
	tenant := insertTenantFixture(t, fmt.Sprintf("bkg%d", time.Now().UnixNano()%1000000), models.TenantProvisionReady)
	uri := fmt.Sprintf("/api/platform/tenants/%d", tenant.ID)

	dir := services.TenantBackupDir(tenant.Code)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	names := []string{"20260101_000001.sql", "20260102_000001.sql", "20260103_000001.sql", "20260104_000001.sql"}
	for _, n := range names {
		require.NoError(t, os.WriteFile(filepath.Join(dir, n), []byte("-- dump "+n), 0o644))
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("not a dump"), 0o644))

	// A restore without a file name is rejected before anything is queued.
	resp, content := platformCall(t, token, "POST", uri+"/restore", `{}`)
	resp.AssertStatus(400)
	assert.Contains(t, content, "invalid_argument")
	var stored models.Tenant
	require.NoError(t, appfacades.PlatformOrmQuery(nil).Where("id", tenant.ID).First(&stored))
	assert.NotEqual(t, models.TenantOpStatusQueued, stored.LastOpStatus)

	// Download only serves .sql files from this tenant's own backup directory.
	for _, name := range []string{"../../../.env", "..\\..\\.env", "notes.txt", "missing.sql"} {
		resp, content = platformCall(t, token, "GET", uri+"/backups/download?name="+url.QueryEscape(name), "")
		assert.False(t, resp.IsSuccessful(), "download %q must be refused", name)
		assert.Contains(t, content, "invalid_argument", name)
	}
	resp, content = platformCall(t, token, "GET", uri+"/backups/download?name="+names[0], "")
	assert.True(t, resp.IsSuccessful(), content)
	assert.Equal(t, "-- dump "+names[0], content)

	// Prune keeps the newest N dumps and ignores non-dump files.
	resp, content = platformCall(t, token, "POST", uri+"/backups/prune", `{"keep":2}`)
	require.True(t, resp.IsSuccessful(), content)
	var pruned struct {
		Data struct {
			Removed int `json:"removed"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(content), &pruned))
	assert.Equal(t, 2, pruned.Data.Removed)
	for i, n := range names {
		_, err := os.Stat(filepath.Join(dir, n))
		assert.Equal(t, i >= 2, err == nil, "%s kept=%v", n, i >= 2)
	}
	_, err := os.Stat(filepath.Join(dir, "notes.txt"))
	assert.NoError(t, err)
}

// hostGate makes a tenant-side request as if it arrived on host (no explicit tenant hint).
func hostGate(t *testing.T, host string) string {
	t.Helper()
	testCase := tests.TestCase{}
	resp, err := testCase.Http(t).
		WithHeader("Authorization", "Bearer unused").
		WithHeader("X-Forwarded-Host", host).
		Get("/api/admin/info")
	require.NoError(t, err)
	content, err := resp.Content()
	require.NoError(t, err)
	return content
}

// tlsAllowed asks the public on-demand TLS endpoint whether a certificate may be issued for host.
func tlsAllowed(t *testing.T, host string) bool {
	t.Helper()
	testCase := tests.TestCase{}
	resp, err := testCase.Http(t).Get("/api/platform/public/tls-allow?host=" + url.QueryEscape(host))
	require.NoError(t, err)
	return resp.IsSuccessful()
}

func createDomain(t *testing.T, token string, tenantID uint, body string) (bool, uint, string) {
	t.Helper()
	resp, content := platformCall(t, token, "POST", fmt.Sprintf("/api/platform/tenants/%d/domains", tenantID), body)
	if !resp.IsSuccessful() {
		return false, 0, content
	}
	var payload struct {
		Data struct {
			Domain struct {
				ID uint `json:"id"`
			} `json:"domain"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(content), &payload))
	return true, payload.Data.Domain.ID, content
}

// A custom domain routes to its tenant only while active, and stops routing the moment it
// is disabled. The tenant is in maintenance, so a host that resolves to it answers
// tenant_maintenance without touching a database; any other answer means it did not resolve.
func TestPlatformTenantCustomDomainLifecycle(t *testing.T) {
	withTenancyDriver(t, "database")
	token := loginPlatformSmoke(t)
	nonce := time.Now().UnixNano() % 1000000
	tenant := insertTenantFixture(t, fmt.Sprintf("dom%d", nonce), models.TenantProvisionReady)
	other := insertTenantFixture(t, fmt.Sprintf("domb%d", nonce), models.TenantProvisionReady)
	host := fmt.Sprintf("crm-%d.example.invalid", nonce) // .invalid never resolves (RFC 2606)
	domainSvc := services.NewTenantDomainService()
	t.Cleanup(func() {
		_ = domainSvc.PurgeAllForTenant(tenant.ID)
		_ = domainSvc.PurgeAllForTenant(other.ID)
	})
	adminSvc := services.NewTenantAdminService()
	_, err := adminSvc.SetMaintenance(tenant.ID, true, "")
	require.NoError(t, err)

	domainsURI := fmt.Sprintf("/api/platform/tenants/%d/domains", tenant.ID)
	for _, body := range []string{`{"host":"bad host"}`, fmt.Sprintf(`{"host":%q,"ssl_mode":"weird"}`, host)} {
		ok, _, content := createDomain(t, token, tenant.ID, body)
		assert.False(t, ok, body)
		assert.Contains(t, content, "tenant_domain_invalid", body)
	}

	ok, domainID, content := createDomain(t, token, tenant.ID, fmt.Sprintf(`{"host":%q,"ssl_mode":"edge"}`, host))
	require.True(t, ok, content)
	domainURI := fmt.Sprintf("%s/%d", domainsURI, domainID)

	// Pending: no routing, no certificate.
	assert.NotContains(t, hostGate(t, host), "tenant_maintenance")
	assert.False(t, tlsAllowed(t, host))

	resp, content := platformCall(t, token, "POST", domainURI+"/verify", "")
	assert.False(t, resp.IsSuccessful())
	assert.Contains(t, content, "tenant_domain_verify_failed")
	var row models.TenantDomain
	require.NoError(t, appfacades.PlatformOrmQuery(nil).Where("id", domainID).First(&row))
	assert.Equal(t, models.TenantDomainStatusPending, row.Status)
	assert.NotEmpty(t, row.LastCheckError, "the DNS failure reason is kept for the console")

	resp, content = platformCall(t, token, "PUT", domainURI+"/primary", "")
	assert.False(t, resp.IsSuccessful())
	assert.Contains(t, content, "tenant_domain_not_active")

	// Simulate a successful DNS check (Verify sets active and clears the host cache).
	_, err = appfacades.PlatformOrmQuery(nil).Model(&models.TenantDomain{}).Where("id", domainID).
		Update("status", models.TenantDomainStatusActive)
	require.NoError(t, err)
	facades.Cache().Forget("tenant_domain:v1:" + host)

	assert.Contains(t, hostGate(t, host), "tenant_maintenance", "active domain routes to its tenant")
	assert.False(t, tlsAllowed(t, host), "no certificate while the tenant is in maintenance")
	_, err = adminSvc.SetMaintenance(tenant.ID, false, "")
	require.NoError(t, err)
	assert.True(t, tlsAllowed(t, host))

	resp, content = platformCall(t, token, "PUT", domainURI+"/primary", "")
	require.True(t, resp.IsSuccessful(), content)

	ok, _, content = createDomain(t, token, other.ID, fmt.Sprintf(`{"host":%q}`, host))
	assert.False(t, ok)
	assert.Contains(t, content, "tenant_domain_taken")

	_, err = adminSvc.SetMaintenance(tenant.ID, true, "")
	require.NoError(t, err)
	// Domain is in use (host cache warm) when the operator disables it.
	require.Contains(t, hostGate(t, host), "tenant_maintenance")
	resp, content = platformCall(t, token, "PUT", domainURI+"/disable", "")
	require.True(t, resp.IsSuccessful(), content)
	assert.NotContains(t, hostGate(t, host), "tenant_maintenance", "a disabled domain stops routing immediately")
	assert.False(t, tlsAllowed(t, host))

	resp, content = platformCall(t, token, "DELETE", domainURI, "")
	require.True(t, resp.IsSuccessful(), content)
	ok, _, content = createDomain(t, token, other.ID, fmt.Sprintf(`{"host":%q}`, host))
	assert.True(t, ok, "a deleted domain can be bound by another tenant: %s", content)
}

func tenantConfigValue(t *testing.T, conn *services.TenantConnectionService, tn *models.Tenant, key string) (string, bool) {
	t.Helper()
	var row models.Config
	err := conn.WithTenantConnection(tn, func() error {
		bound := tenancyctx.WithTenant(context.Background(), tn.ID, tn.ConnectionName, tn.Code)
		return appfacades.OrmQuery(bound).Where("key", key).First(&row)
	})
	require.NoError(t, err)
	return row.Value, row.ID > 0
}

func execTenantSQL(t *testing.T, conn *services.TenantConnectionService, tn *models.Tenant, fn func(ctx context.Context) error) {
	t.Helper()
	require.NoError(t, conn.WithTenantConnection(tn, func() error {
		return fn(tenancyctx.WithTenant(context.Background(), tn.ID, tn.ConnectionName, tn.Code))
	}))
}

// Backup -> change data -> restore puts the tenant database back to the backup.
// Runs the same RunTenantOp the queue job runs, so nothing is left on a real queue.
func TestTenantBackupThenRestoreRevertsData(t *testing.T) {
	for _, bin := range []string{"mysqldump", "mysql"} {
		if _, err := exec.LookPath(bin); err != nil {
			t.Skipf("skip backup/restore: %s not on PATH", bin)
		}
	}
	withTenancyDriver(t, "database")
	prevAllow := facades.Config().GetBool("tenancy.allow_platform_db_credentials", false)
	facades.Config().Add("tenancy.allow_platform_db_credentials", true)
	t.Cleanup(func() { facades.Config().Add("tenancy.allow_platform_db_credentials", prevAllow) })
	appfacades.InstallTenantAwareSchema(facades.App())
	appfacades.InstallTenantAwareOrm(facades.App())
	if def := facades.Config().GetString("database.default", "mysql"); def != "" {
		facades.Schema().SetConnection(def)
	}

	adminSvc := services.NewTenantAdminService()
	conn := services.NewTenantConnectionService()
	code := fmt.Sprintf("rst%d", time.Now().UnixNano()%1_000_000_000)
	tn, err := adminSvc.Create(services.TenantCreateInput{Code: code, Name: "Restore " + code, Migrate: true})
	if err != nil {
		t.Skipf("skip backup/restore: cannot create/migrate tenant DB: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(services.TenantBackupDir(tn.Code))
		_, _ = appfacades.PlatformOrmQuery(nil).Where("tenant_id", tn.ID).Delete(&models.TenantOpLog{})
		_ = conn.DropStorage(tn)
		_, _ = appfacades.PlatformOrmQuery(nil).Where("id", tn.ID).ForceDelete(&models.Tenant{})
		if def := facades.Config().GetString("database.default", "mysql"); def != "" {
			facades.Schema().SetConnection(def)
		}
	})
	require.Equal(t, models.TenantProvisionReady, tn.ProvisionStatus)

	execTenantSQL(t, conn, tn, func(ctx context.Context) error {
		return appfacades.OrmQuery(ctx).Create(&models.Config{Group: "test", Key: "restore_marker", Value: "before"})
	})

	ops := services.NewTenantOpsService()
	actor := services.TenantOpActor{Name: "test:restore"}
	_, backupArgs, err := ops.BeginQueuedBackup(tn.ID, -1, actor, "")
	require.NoError(t, err)
	require.NoError(t, services.RunTenantOp(backupArgs))

	files, _, err := services.ListTenantBackups(tn)
	require.NoError(t, err)
	require.Len(t, files, 1)
	backupName := files[0].Name

	execTenantSQL(t, conn, tn, func(ctx context.Context) error {
		if _, err := appfacades.OrmQuery(ctx).Model(&models.Config{}).Where("key", "restore_marker").Update("value", "after"); err != nil {
			return err
		}
		return appfacades.OrmQuery(ctx).Create(&models.Config{Group: "test", Key: "added_after_backup", Value: "x"})
	})
	value, _ := tenantConfigValue(t, conn, tn, "restore_marker")
	require.Equal(t, "after", value)

	_, restoreArgs, err := ops.BeginQueuedOpWithBackup(tn.ID, models.TenantOpRestore, false, backupName, actor, "")
	require.NoError(t, err)
	require.NoError(t, services.RunTenantOp(restoreArgs))

	value, found := tenantConfigValue(t, conn, tn, "restore_marker")
	assert.True(t, found)
	assert.Equal(t, "before", value, "restore brings back the backed-up value")
	_, found = tenantConfigValue(t, conn, tn, "added_after_backup")
	assert.False(t, found, "rows written after the backup are gone")

	fresh, err := adminSvc.GetByID(tn.ID)
	require.NoError(t, err)
	assert.Equal(t, models.TenantOpRestore, fresh.LastOp)
	assert.Equal(t, models.TenantOpStatusSuccess, fresh.LastOpStatus, fresh.LastOpMessage)
}
