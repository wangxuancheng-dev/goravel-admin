package feature_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

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

	testCase := tests.TestCase{}
	captchaResp, err := testCase.Http(t).Get("/api/platform/login/captcha")
	require.NoError(t, err)
	captchaResp.AssertOk()
	captchaContent, err := captchaResp.Content()
	require.NoError(t, err)

	var captchaPayload struct {
		Code int `json:"code"`
		Data struct {
			Captcha struct {
				CaptchaID string `json:"captcha_id"`
			} `json:"captcha"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(captchaContent), &captchaPayload))
	require.Equal(t, 200, captchaPayload.Code)
	captchaID := captchaPayload.Data.Captcha.CaptchaID
	require.NotEmpty(t, captchaID)
	answer := services.PeekCaptchaAnswer(captchaID)
	require.NotEmpty(t, answer)

	body := fmt.Sprintf(
		`{"username":%q,"password":%q,"captcha_id":%q,"captcha_answer":%q}`,
		platformSmokeUser, platformSmokePass, captchaID, answer,
	)
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

func TestPlatformLoginLogAndOperationLog(t *testing.T) {
	withTenancyDriver(t, "database")
	token := loginPlatformSmoke(t)
	testCase := tests.TestCase{}

	loginListResp, err := testCase.Http(t).
		WithHeader("Authorization", "Bearer "+token).
		Get("/api/platform/login-logs?username=" + platformSmokeUser)
	require.NoError(t, err)
	loginListResp.AssertOk()
	loginBody, err := loginListResp.Content()
	require.NoError(t, err)
	assert.Contains(t, loginBody, platformSmokeUser)
	assert.Contains(t, loginBody, "login_success")

	// wrong password -> failed login log
	captchaResp, err := testCase.Http(t).Get("/api/platform/login/captcha")
	require.NoError(t, err)
	captchaResp.AssertOk()
	captchaContent, err := captchaResp.Content()
	require.NoError(t, err)
	var captchaPayload struct {
		Data struct {
			Captcha struct {
				CaptchaID string `json:"captcha_id"`
			} `json:"captcha"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(captchaContent), &captchaPayload))
	captchaID := captchaPayload.Data.Captcha.CaptchaID
	answer := services.PeekCaptchaAnswer(captchaID)
	require.NotEmpty(t, answer)
	failBody := fmt.Sprintf(
		`{"username":%q,"password":"wrong-password","captcha_id":%q,"captcha_answer":%q}`,
		platformSmokeUser, captchaID, answer,
	)
	failResp, err := testCase.Http(t).
		WithHeader("Content-Type", "application/json").
		Post("/api/platform/login", strings.NewReader(failBody))
	require.NoError(t, err)
	failContent, err := failResp.Content()
	require.NoError(t, err)
	assert.Contains(t, failContent, "username_or_password_error")

	failListResp, err := testCase.Http(t).
		WithHeader("Authorization", "Bearer "+token).
		Get("/api/platform/login-logs?username=" + platformSmokeUser + "&status=0")
	require.NoError(t, err)
	failListResp.AssertOk()
	failListBody, err := failListResp.Content()
	require.NoError(t, err)
	assert.Contains(t, failListBody, "password_error")

	// write op -> operation log
	pwdBody := fmt.Sprintf(
		`{"old_password":%q,"new_password":%q,"confirm_password":%q}`,
		platformSmokePass,
		platformSmokePass+"Y",
		platformSmokePass+"Y",
	)
	pwdResp, err := testCase.Http(t).
		WithHeader("Authorization", "Bearer "+token).
		WithHeader("Content-Type", "application/json").
		Put("/api/platform/password", strings.NewReader(pwdBody))
	require.NoError(t, err)
	pwdResp.AssertOk()
	_, err = services.UpsertPlatformAdmin(platformSmokeUser, platformSmokePass, "Smoke Platform", "owner")
	require.NoError(t, err)

	opListResp, err := testCase.Http(t).
		WithHeader("Authorization", "Bearer "+token).
		Get("/api/platform/operation-logs?path=/api/platform/password")
	require.NoError(t, err)
	opListResp.AssertOk()
	opBody, err := opListResp.Content()
	require.NoError(t, err)
	assert.Contains(t, opBody, "platform.password.update")
	assert.Contains(t, opBody, platformSmokeUser)
	assert.Contains(t, opBody, "***")
}

func TestPlatformAdminCRUD(t *testing.T) {
	withTenancyDriver(t, "database")
	token := loginPlatformSmoke(t)
	testCase := tests.TestCase{}

	listResp, err := testCase.Http(t).
		WithHeader("Authorization", "Bearer "+token).
		Get("/api/platform/admins")
	require.NoError(t, err)
	listResp.AssertOk()
	listBody, err := listResp.Content()
	require.NoError(t, err)
	assert.Contains(t, listBody, platformSmokeUser)

	viewerUser := fmt.Sprintf("smoke_platform_viewer_%d", time.Now().UnixNano()%1000000)
	viewerPass := "SmokeViewer123!"
	createBody := fmt.Sprintf(
		`{"username":%q,"password":%q,"name":"Viewer","role":"viewer"}`,
		viewerUser, viewerPass,
	)
	createResp, err := testCase.Http(t).
		WithHeader("Authorization", "Bearer "+token).
		WithHeader("Content-Type", "application/json").
		Post("/api/platform/admins", strings.NewReader(createBody))
	require.NoError(t, err)
	createResp.AssertOk()
	createContent, err := createResp.Content()
	require.NoError(t, err)

	var created struct {
		Code int `json:"code"`
		Data struct {
			Admin struct {
				ID       uint   `json:"id"`
				Username string `json:"username"`
				Role     string `json:"role"`
			} `json:"admin"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(createContent), &created))
	require.Equal(t, 200, created.Code)
	require.NotZero(t, created.Data.Admin.ID)
	assert.Equal(t, "viewer", created.Data.Admin.Role)
	viewerID := created.Data.Admin.ID

	// cannot demote/disable self
	selfID := uint(0)
	infoResp, err := testCase.Http(t).
		WithHeader("Authorization", "Bearer "+token).
		Get("/api/platform/info")
	require.NoError(t, err)
	infoBody, err := infoResp.Content()
	require.NoError(t, err)
	var infoPayload struct {
		Data struct {
			Admin struct {
				ID uint `json:"id"`
			} `json:"admin"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(infoBody), &infoPayload))
	selfID = infoPayload.Data.Admin.ID
	require.NotZero(t, selfID)

	badSelf := `{"role":"viewer"}`
	selfResp, err := testCase.Http(t).
		WithHeader("Authorization", "Bearer "+token).
		WithHeader("Content-Type", "application/json").
		Put(fmt.Sprintf("/api/platform/admins/%d", selfID), strings.NewReader(badSelf))
	require.NoError(t, err)
	selfContent, err := selfResp.Content()
	require.NoError(t, err)
	assert.Contains(t, selfContent, "platform_cannot_modify_self")

	resetBody := `{"password":"SmokeViewer123!X","confirm_password":"SmokeViewer123!X"}`
	resetResp, err := testCase.Http(t).
		WithHeader("Authorization", "Bearer "+token).
		WithHeader("Content-Type", "application/json").
		Post(fmt.Sprintf("/api/platform/admins/%d/reset-password", viewerID), strings.NewReader(resetBody))
	require.NoError(t, err)
	resetResp.AssertOk()

	delResp, err := testCase.Http(t).
		WithHeader("Authorization", "Bearer "+token).
		Delete(fmt.Sprintf("/api/platform/admins/%d", viewerID), strings.NewReader(""))
	require.NoError(t, err)
	delResp.AssertOk()

	// cannot delete self
	delSelfResp, err := testCase.Http(t).
		WithHeader("Authorization", "Bearer "+token).
		Delete(fmt.Sprintf("/api/platform/admins/%d", selfID), strings.NewReader(""))
	require.NoError(t, err)
	delSelfBody, err := delSelfResp.Content()
	require.NoError(t, err)
	assert.Contains(t, delSelfBody, "admin_cannot_delete_self")
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
