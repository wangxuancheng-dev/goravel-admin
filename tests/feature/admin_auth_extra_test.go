package feature_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"goravel/tests"
)

func TestAdminLoginWrongPassword(t *testing.T) {
	ensureSmokeAdmin(t)

	body := fmt.Sprintf(`{"username":%q,"password":%q}`, smokeAdminUsername, "definitely-wrong-password")
	testCase := tests.TestCase{}
	resp, err := testCase.Http(t).
		WithHeader("Content-Type", "application/json").
		Post("/api/admin/login", strings.NewReader(body))
	require.NoError(t, err)

	content, err := resp.Content()
	require.NoError(t, err)

	var payload struct {
		Code      int    `json:"code"`
		ErrorCode string `json:"error_code"`
	}
	require.NoError(t, json.Unmarshal([]byte(content), &payload))
	assert.NotEqual(t, 200, payload.Code)
	assert.True(t,
		payload.ErrorCode == "username_or_password_error" ||
			payload.ErrorCode == "password_error" ||
			payload.ErrorCode == "login_failed" ||
			payload.ErrorCode == "captcha_required" ||
			strings.Contains(content, "username_or_password") ||
			strings.Contains(content, "password"),
		"unexpected login failure body: %s", content,
	)
}

func TestAdminResourceForbiddenWithoutPermission(t *testing.T) {
	token := loginSmokeAdmin(t)

	testCase := tests.TestCase{}
	resp, err := testCase.Http(t).
		WithHeader("Authorization", "Bearer "+token).
		Get("/api/admin/admins")
	require.NoError(t, err)

	content, err := resp.Content()
	require.NoError(t, err)
	assert.Contains(t, content, "no_permission")
}
