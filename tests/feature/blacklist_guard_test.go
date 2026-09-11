package feature_test

import (
	"testing"

	"github.com/dromara/carbon/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/tests"
)

func TestAdminBlockedByBlacklistIP(t *testing.T) {
	services.ResetBlacklistCacheForTest()
	t.Cleanup(services.ResetBlacklistCacheForTest)

	// Http test client uses documentation IP 192.0.2.1
	blockedIP := "192.0.2.1"
	row := &models.Blacklist{}
	err := appfacades.OrmQuery(nil).Model(row).Create(map[string]any{
		"ip":         blockedIP,
		"remark":     "feature test block",
		"status":     uint8(1),
		"created_at": carbon.Now(),
		"updated_at": carbon.Now(),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = appfacades.OrmQuery(nil).Where("ip", blockedIP).Where("remark", "feature test block").Delete(&models.Blacklist{})
		services.InvalidateBlacklistCache(nil)
	})
	services.InvalidateBlacklistCache(nil)

	token := loginSmokeAdmin(t)
	testCase := tests.TestCase{}
	resp, err := testCase.Http(t).
		WithHeader("Authorization", "Bearer "+token).
		Get("/api/admin/info")
	require.NoError(t, err)

	content, err := resp.Content()
	require.NoError(t, err)
	assert.Contains(t, content, "ip_blocked")
}

func TestPlatformInfoUnauthorized(t *testing.T) {
	withTenancyDriver(t, "database")

	testCase := tests.TestCase{}
	resp, err := testCase.Http(t).Get("/api/platform/info")
	require.NoError(t, err)

	content, err := resp.Content()
	require.NoError(t, err)
	assert.True(t,
		containsAny(content, "not_logged_in", "unauthorized", "token_required", "invalid_token"),
		"unexpected body: %s", content,
	)
}

func containsAny(haystack string, needles ...string) bool {
	for _, n := range needles {
		if assert.ObjectsAreEqual(true, true) && len(n) > 0 {
			for i := 0; i+len(n) <= len(haystack); i++ {
				if haystack[i:i+len(n)] == n {
					return true
				}
			}
		}
	}
	return false
}
