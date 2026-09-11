package feature_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/goravel/framework/facades"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"goravel/app/models"
	"goravel/tests"
)

func TestOrdersForbiddenWithoutPermission(t *testing.T) {
	token := loginSmokeAdmin(t)

	testCase := tests.TestCase{}
	resp, err := testCase.Http(t).
		WithHeader("Authorization", "Bearer "+token).
		Get("/api/admin/orders")
	require.NoError(t, err)

	content, err := resp.Content()
	require.NoError(t, err)
	assert.Contains(t, content, "no_permission")
}

func TestExportsForbiddenWithoutPermission(t *testing.T) {
	token := loginSmokeAdmin(t)

	testCase := tests.TestCase{}
	resp, err := testCase.Http(t).
		WithHeader("Authorization", "Bearer "+token).
		Get("/api/admin/exports")
	require.NoError(t, err)

	content, err := resp.Content()
	require.NoError(t, err)
	assert.Contains(t, content, "no_permission")
}

func TestExportDownloadForbiddenForNonOwner(t *testing.T) {
	owner := ensureNamedAdmin(t, "export_owner_admin", "ExportOwner123!")
	actorToken := loginNamedAdminWithExportDownload(t)

	rec := models.Export{
		AdminID:   owner.ID,
		Type:      models.ExportTypeAdmins,
		Disk:      "local",
		Path:      "exports/ownership-test.csv",
		Filename:  "ownership-test.csv",
		Extension: "csv",
		Size:      12,
		Status:    models.ExportStatusSuccess,
	}
	require.NoError(t, facades.Orm().Query().Create(&rec))
	t.Cleanup(func() {
		_, _ = facades.Orm().Query().Delete(&rec)
	})

	testCase := tests.TestCase{}
	resp, err := testCase.Http(t).
		WithHeader("Authorization", "Bearer "+actorToken).
		Get(fmt.Sprintf("/api/admin/exports/%d/download", rec.ID))
	require.NoError(t, err)

	content, err := resp.Content()
	require.NoError(t, err)

	var payload struct {
		Code      int    `json:"code"`
		ErrorCode string `json:"error_code"`
		Message   string `json:"message"`
	}
	require.NoError(t, json.Unmarshal([]byte(content), &payload))
	assert.Equal(t, 403, payload.Code)
	assert.True(t,
		payload.ErrorCode == "forbidden" ||
			strings.Contains(content, "forbidden") ||
			strings.Contains(payload.Message, "forbidden"),
		"unexpected body: %s", content,
	)
}

func ensureNamedAdmin(t *testing.T, username, password string) models.Admin {
	t.Helper()

	var admin models.Admin
	err := facades.Orm().Query().Where("username", username).First(&admin)
	if err == nil && admin.ID > 0 {
		return admin
	}

	hashed, err := facades.Hash().Make(password)
	require.NoError(t, err)
	admin = models.Admin{
		Username: username,
		Password: hashed,
		Nickname: username,
		Status:   1,
	}
	require.NoError(t, facades.Orm().Query().Create(&admin))
	return admin
}

func loginNamedAdminWithExportDownload(t *testing.T) string {
	t.Helper()

	const (
		username = "export_actor_admin"
		password = "ExportActor123!"
		roleSlug = "export_download_only"
	)

	admin := ensureNamedAdmin(t, username, password)

	var role models.Role
	err := facades.Orm().Query().Where("slug", roleSlug).First(&role)
	if err != nil || role.ID == 0 {
		role = models.Role{
			Name:   "Export Download Only",
			Slug:   roleSlug,
			Status: 1,
			Sort:   999,
		}
		require.NoError(t, facades.Orm().Query().Create(&role))
	}

	var perm models.Permission
	err = facades.Orm().Query().Where("slug", "export.download").First(&perm)
	if err != nil || perm.ID == 0 {
		perm = models.Permission{
			Name:   "导出数据下载",
			Slug:   "export.download",
			Method: "GET",
			Path:   "/api/admin/exports/*/download",
			Status: 1,
			Sort:   2,
		}
		require.NoError(t, facades.Orm().Query().Create(&perm))
	}

	require.NoError(t, facades.Orm().Query().Model(&role).Association("Permissions").Replace([]models.Permission{perm}))
	require.NoError(t, facades.Orm().Query().Model(&admin).Association("Roles").Replace([]models.Role{role}))

	// Avoid reusing smoke token cache: login this actor fresh each call is fine for one test.
	body := fmt.Sprintf(`{"username":%q,"password":%q}`, username, password)
	testCase := tests.TestCase{}
	resp, err := testCase.Http(t).
		WithHeader("Content-Type", "application/json").
		Post("/api/admin/login", strings.NewReader(body))
	require.NoError(t, err)
	content, err := resp.Content()
	require.NoError(t, err)

	var payload struct {
		Code int `json:"code"`
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(content), &payload))
	require.Equal(t, 200, payload.Code, "login body: %s", content)
	require.NotEmpty(t, payload.Data.Token)
	return payload.Data.Token
}
