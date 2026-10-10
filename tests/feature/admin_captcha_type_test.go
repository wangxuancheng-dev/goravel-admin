package feature_test

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"

	"github.com/goravel/framework/facades"
	"github.com/stretchr/testify/require"

	apperrors "goravel/app/errors"
	"goravel/app/services"
	"goravel/tests"
)

type adminCaptchaPayload struct {
	Code int `json:"code"`
	Data struct {
		Captcha map[string]any `json:"captcha"`
	} `json:"data"`
}

func fetchAdminLoginCaptcha(t *testing.T) map[string]any {
	t.Helper()
	testCase := tests.TestCase{}
	resp, err := testCase.Http(t).Get("/api/admin/login/captcha")
	require.NoError(t, err)
	content, err := resp.Content()
	require.NoError(t, err)

	var payload adminCaptchaPayload
	require.NoError(t, json.Unmarshal([]byte(content), &payload), content)
	require.Equal(t, 200, payload.Code, content)
	return payload.Data.Captcha
}

// Tenant admin login captcha type is a per-tenant DB config (captcha.captcha_type).
func TestAdminLoginCaptchaTypeFollowsConfig(t *testing.T) {
	withTenancyDriver(t, "off")
	cfg := services.NewConfigService(context.Background())

	// Remember current values so the shared test DB is left as found.
	prev := map[string]string{"captcha_enabled": "0", "captcha_type": "image"}
	rows, err := cfg.GetByGroup("captcha")
	require.NoError(t, err)
	for _, row := range rows {
		if _, ok := prev[row.Key]; ok {
			prev[row.Key] = row.Value
		}
	}
	t.Cleanup(func() {
		_ = cfg.Save("captcha", map[string]any{
			"captcha_enabled": prev["captcha_enabled"],
			"captcha_type":    prev["captcha_type"],
		})
	})

	// Unsupported types are rejected.
	err = cfg.Save("captcha", map[string]any{"captcha_type": "rotate"})
	require.ErrorIs(t, err, apperrors.ErrCaptchaTypeInvalid)

	// Slide.
	require.NoError(t, cfg.Save("captcha", map[string]any{"captcha_enabled": "1", "captcha_type": "slide"}))
	info := fetchAdminLoginCaptcha(t)
	require.Equal(t, "slide", info["type"])
	require.NotEmpty(t, info["captcha_id"])
	require.NotEmpty(t, info["master_image"])
	require.NotEmpty(t, info["tile_image"])
	require.Empty(t, info["captcha_image"])

	id, _ := info["captcha_id"].(string)
	answer := services.PeekCaptchaAnswer(id)
	x, err := strconv.Atoi(answer)
	require.NoError(t, err, answer)
	ok, key := services.NewCaptchaServiceImpl(context.Background()).Verify(id, strconv.Itoa(x))
	require.True(t, ok, key)

	// Image (default).
	require.NoError(t, cfg.Save("captcha", map[string]any{"captcha_enabled": "1", "captcha_type": "image"}))
	info = fetchAdminLoginCaptcha(t)
	require.Equal(t, "image", info["type"])
	require.NotEmpty(t, info["captcha_image"])
	require.Empty(t, info["master_image"])
}

// Platform captcha: check=1 only reports the configured type and does not issue a challenge.
func TestPlatformLoginCaptchaCheckOnlyReportsType(t *testing.T) {
	withTenancyDriver(t, "database")
	withTenancyDriver(t, "database")
	testCase := tests.TestCase{}
	resp, err := testCase.Http(t).Get("/api/platform/login/captcha?check=1")
	require.NoError(t, err)
	content, err := resp.Content()
	require.NoError(t, err)

	var payload adminCaptchaPayload
	require.NoError(t, json.Unmarshal([]byte(content), &payload), content)
	require.Equal(t, 200, payload.Code, content)
	want := services.NormalizeCaptchaType(facades.Config().GetString("tenancy.platform_captcha_type", "image"))
	require.Equal(t, want, payload.Data.Captcha["type"])
	require.Empty(t, payload.Data.Captcha["captcha_id"])
}
