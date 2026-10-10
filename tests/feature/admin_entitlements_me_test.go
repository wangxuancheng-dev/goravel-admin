package feature_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"goravel/tests"
)

// With tenancy off, the tenant-side plan summary reports enabled=false.
func TestAdminEntitlementsMeDisabledWithoutTenancy(t *testing.T) {
	token := loginSmokeAdmin(t)

	testCase := tests.TestCase{}
	resp, err := testCase.Http(t).
		WithHeader("Authorization", "Bearer "+token).
		Get("/api/admin/entitlements/me")
	require.NoError(t, err)
	resp.AssertOk()

	content, err := resp.Content()
	require.NoError(t, err)

	var payload struct {
		Code int `json:"code"`
		Data struct {
			Enabled bool `json:"enabled"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(content), &payload))
	require.Equal(t, 200, payload.Code)
	require.False(t, payload.Data.Enabled)
}
