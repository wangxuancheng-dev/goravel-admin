package feature_test

import (
	"context"
	"testing"
	"time"

	"github.com/goravel/framework/facades"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"goravel/app/models"
	"goravel/app/services"
	"goravel/tests"
)

// A valid token whose admin row is gone is a 401, not "account disabled": ORM First does not
// error when nothing matches, so the middleware must check the loaded ID.
func TestAdminJwtTokenForMissingAdmin(t *testing.T) {
	const missingAdminID = 3999999999
	token, _, err := services.NewTokenServiceImpl(context.Background()).
		CreateToken("admin", missingAdminID, "jwt-missing-admin", nil, "", "", "", "")
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = facades.Orm().Query().Model(&models.PersonalAccessToken{}).
			Where("tokenable_type", "admin").Where("tokenable_id", missingAdminID).Delete()
	})

	testCase := tests.TestCase{}
	resp, err := testCase.Http(t).
		WithHeader("Authorization", "Bearer "+token).
		Get("/api/admin/info")
	require.NoError(t, err)
	resp.AssertUnauthorized()
	content, err := resp.Content()
	require.NoError(t, err)
	assert.NotContains(t, content, "account_disabled")
}

// Requests refresh last_used_at and the sliding expires_at at most once a minute.
func TestAdminJwtTouchIsThrottled(t *testing.T) {
	ensureSmokeAdmin(t)
	var admin models.Admin
	require.NoError(t, facades.Orm().Query().Where("username", smokeAdminUsername).FirstOrFail(&admin))

	shortExpiry := time.Now().Add(10 * time.Minute)
	token, created, err := services.NewTokenServiceImpl(context.Background()).
		CreateToken("admin", admin.ID, "jwt-touch", &shortExpiry, "", "", "", "")
	require.NoError(t, err)
	require.NotNil(t, created)
	t.Cleanup(func() {
		_, _ = facades.Orm().Query().Model(&models.PersonalAccessToken{}).Where("id", created.ID).Delete()
	})

	load := func() models.PersonalAccessToken {
		var row models.PersonalAccessToken
		require.NoError(t, facades.Orm().Query().Where("id", created.ID).FirstOrFail(&row))
		require.NotNil(t, row.LastUsedAt)
		require.NotNil(t, row.ExpiresAt)
		return row
	}
	call := func() {
		testCase := tests.TestCase{}
		resp, callErr := testCase.Http(t).
			WithHeader("Authorization", "Bearer "+token).
			Get("/api/admin/info")
		require.NoError(t, callErr)
		resp.AssertOk()
	}

	before := load()
	call()
	after := load()
	assert.True(t, after.LastUsedAt.Equal(*before.LastUsedAt), "used just now: last_used_at is not rewritten")
	assert.True(t, after.ExpiresAt.Equal(*before.ExpiresAt), "used just now: expires_at is not rewritten")

	_, err = facades.Orm().Query().Model(&models.PersonalAccessToken{}).Where("id", created.ID).
		Update("last_used_at", time.Now().Add(-2*time.Minute))
	require.NoError(t, err)
	call()
	touched := load()
	assert.WithinDuration(t, time.Now(), *touched.LastUsedAt, 5*time.Second)
	ttl := time.Duration(facades.Config().GetInt("jwt.ttl", 60)) * time.Minute
	assert.WithinDuration(t, time.Now().Add(ttl), *touched.ExpiresAt, 5*time.Second)
}

func TestAdminJwtUnknownToken(t *testing.T) {
	testCase := tests.TestCase{}
	resp, err := testCase.Http(t).
		WithHeader("Authorization", "Bearer not-a-real-token").
		Get("/api/admin/info")
	require.NoError(t, err)
	resp.AssertUnauthorized()
}
