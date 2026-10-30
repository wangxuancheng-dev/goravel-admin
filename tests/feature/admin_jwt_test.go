package feature_test

import (
	"context"
	"testing"

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

func TestAdminJwtUnknownToken(t *testing.T) {
	testCase := tests.TestCase{}
	resp, err := testCase.Http(t).
		WithHeader("Authorization", "Bearer not-a-real-token").
		Get("/api/admin/info")
	require.NoError(t, err)
	resp.AssertUnauthorized()
}
