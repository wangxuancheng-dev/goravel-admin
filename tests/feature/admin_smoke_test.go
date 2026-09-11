package feature_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/goravel/framework/facades"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"goravel/app/models"
	"goravel/tests"
)

const (
	smokeAdminUsername = "smoke_admin"
	smokeAdminPassword = "SmokeAdmin123!"
)

var (
	smokeAdminTokenOnce sync.Once
	smokeAdminToken     string
	smokeAdminTokenErr  error
)

func ensureSmokeAdmin(t *testing.T) {
	t.Helper()

	exists, err := facades.Orm().Query().Model(&models.Admin{}).Where("username", smokeAdminUsername).Exists()
	require.NoError(t, err)
	if exists {
		return
	}

	hashed, err := facades.Hash().Make(smokeAdminPassword)
	require.NoError(t, err)

	admin := models.Admin{
		Username: smokeAdminUsername,
		Password: hashed,
		Nickname: "Smoke Admin",
		Status:   1,
	}
	require.NoError(t, facades.Orm().Query().Create(&admin))
}

func loginSmokeAdmin(t *testing.T) string {
	t.Helper()
	ensureSmokeAdmin(t)

	smokeAdminTokenOnce.Do(func() {
		body := fmt.Sprintf(`{"username":%q,"password":%q}`, smokeAdminUsername, smokeAdminPassword)
		testCase := tests.TestCase{}
		resp, err := testCase.Http(t).
			WithHeader("Content-Type", "application/json").
			Post("/api/admin/login", strings.NewReader(body))
		if err != nil {
			smokeAdminTokenErr = err
			return
		}
		content, err := resp.Content()
		if err != nil {
			smokeAdminTokenErr = err
			return
		}
		var payload struct {
			Code int `json:"code"`
			Data struct {
				Token string `json:"token"`
			} `json:"data"`
		}
		if err := json.Unmarshal([]byte(content), &payload); err != nil {
			smokeAdminTokenErr = err
			return
		}
		if payload.Code != 200 || payload.Data.Token == "" {
			smokeAdminTokenErr = fmt.Errorf("smoke admin login failed: %s", content)
			return
		}
		smokeAdminToken = payload.Data.Token
	})
	require.NoError(t, smokeAdminTokenErr)
	require.NotEmpty(t, smokeAdminToken, "login should return token")
	return smokeAdminToken
}

func TestAdminLoginSuccess(t *testing.T) {
	token := loginSmokeAdmin(t)
	assert.NotEmpty(t, token)
}

func TestAdminInfoWithToken(t *testing.T) {
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
		Code int            `json:"code"`
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(content), &payload))
	require.Equal(t, 200, payload.Code)
	require.NotEmpty(t, payload.Data)
}

func TestAdminMenusTreeWithToken(t *testing.T) {
	token := loginSmokeAdmin(t)

	testCase := tests.TestCase{}
	resp, err := testCase.Http(t).
		WithHeader("Authorization", "Bearer "+token).
		Get("/api/admin/menus/tree")
	require.NoError(t, err)
	resp.AssertOk()

	content, err := resp.Content()
	require.NoError(t, err)

	var payload struct {
		Code int `json:"code"`
	}
	require.NoError(t, json.Unmarshal([]byte(content), &payload))
	require.Equal(t, 200, payload.Code)
}
