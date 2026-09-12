package feature_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"goravel/tests"
)

func TestAdminInfoExposesModuleConfig(t *testing.T) {
	token := loginSmokeAdmin(t)

	testCase := tests.TestCase{}
	resp, err := testCase.Http(t).
		WithHeader("Authorization", "Bearer "+token).
		Get("/api/admin/info")
	require.NoError(t, err)
	resp.AssertOk()

	content, err := resp.Content()
	require.NoError(t, err)

	var payload struct {
		Code int `json:"code"`
		Data struct {
			Config map[string]any `json:"config"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(content), &payload))
	require.Equal(t, 200, payload.Code)

	cfg := payload.Data.Config
	require.NotNil(t, cfg)
	for _, key := range []string{
		"ai_enabled",
		"orders_enabled",
		"payments_enabled",
		"payment_gateways",
		"dev_tools_enabled",
		"code_generator_enabled",
		"search_enabled",
		"search_driver",
		"otel_enabled",
	} {
		_, ok := cfg[key]
		assert.True(t, ok, "config should expose %s", key)
	}
}
