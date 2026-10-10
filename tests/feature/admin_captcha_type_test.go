package feature_test

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"testing"

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

// Platform console captcha type is stored in landlord platform_settings and edited via /api/platform/settings.
func TestPlatformSettingsCaptchaType(t *testing.T) {
	withTenancyDriver(t, "database")
	token := loginPlatformSmoke(t)
	testCase := tests.TestCase{}

	t.Cleanup(func() {
		_ = services.SetPlatformCaptchaType(context.Background(), "image")
	})

	put := func(body string) (int, map[string]any) {
		resp, err := testCase.Http(t).
			WithHeader("Authorization", "Bearer "+token).
			WithHeader("Content-Type", "application/json").
			Put("/api/platform/settings", strings.NewReader(body))
		require.NoError(t, err)
		content, err := resp.Content()
		require.NoError(t, err)
		var payload struct {
			Code int `json:"code"`
			Data struct {
				Settings map[string]any `json:"settings"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal([]byte(content), &payload), content)
		return payload.Code, payload.Data.Settings
	}
	checkType := func() any {
		resp, err := testCase.Http(t).Get("/api/platform/login/captcha?check=1")
		require.NoError(t, err)
		content, err := resp.Content()
		require.NoError(t, err)
		var payload adminCaptchaPayload
		require.NoError(t, json.Unmarshal([]byte(content), &payload), content)
		require.Equal(t, 200, payload.Code, content)
		require.Empty(t, payload.Data.Captcha["captcha_id"]) // check=1 never issues a challenge
		return payload.Data.Captcha["type"]
	}

	// Default is the classic text captcha.
	require.NoError(t, services.SetPlatformCaptchaType(context.Background(), "image"))
	require.Equal(t, "image", checkType())

	// Invalid type is rejected and nothing changes.
	code, _ := put(`{"captcha_type":"rotate"}`)
	require.Equal(t, 400, code)
	require.Equal(t, "image", checkType())

	// Switch to slide: check endpoint and a real challenge both follow.
	code, settings := put(`{"captcha_type":"slide"}`)
	require.Equal(t, 200, code)
	require.Equal(t, "slide", settings["captcha_type"])
	require.Equal(t, "slide", checkType())
	challenge, err := services.NewPlatformCaptchaService(context.Background()).GenerateChallenge()
	require.NoError(t, err)
	require.Equal(t, "slide", challenge.Type)
	require.NotEmpty(t, challenge.MasterImage)

	// Switch back.
	code, settings = put(`{"captcha_type":"image"}`)
	require.Equal(t, 200, code)
	require.Equal(t, "image", settings["captcha_type"])
	require.Equal(t, "image", checkType())
}
