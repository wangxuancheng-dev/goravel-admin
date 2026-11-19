package feature_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"goravel/tests"
)

// TestPackageFeedbackCRUD exercises the example Goravel package APIs
// (login required, no Permission middleware / no menu install).
func TestPackageFeedbackCRUD(t *testing.T) {
	token := loginSmokeAdmin(t)
	marker := fmt.Sprintf("pkg-feedback-%d", time.Now().UnixNano())
	testCase := tests.TestCase{}

	createBody := fmt.Sprintf(`{"contact":"demo","content":%q}`, marker)
	resp, err := testCase.Http(t).
		WithHeader("Authorization", "Bearer "+token).
		WithHeader("Content-Type", "application/json").
		Post("/api/admin/package-demos/feedbacks", strings.NewReader(createBody))
	require.NoError(t, err)
	content, err := resp.Content()
	require.NoError(t, err)

	var created struct {
		Code int `json:"code"`
		Data struct {
			Feedback struct {
				ID      uint   `json:"id"`
				Content string `json:"content"`
				Status  uint8  `json:"status"`
			} `json:"feedback"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(content), &created), content)
	require.Equal(t, 200, created.Code, content)
	require.NotZero(t, created.Data.Feedback.ID)
	assert.Equal(t, marker, created.Data.Feedback.Content)
	assert.EqualValues(t, 0, created.Data.Feedback.Status)
	id := created.Data.Feedback.ID

	resp, err = testCase.Http(t).
		WithHeader("Authorization", "Bearer "+token).
		Get(fmt.Sprintf("/api/admin/package-demos/feedbacks/%d", id))
	require.NoError(t, err)
	content, err = resp.Content()
	require.NoError(t, err)
	assert.Contains(t, content, marker)

	resp, err = testCase.Http(t).
		WithHeader("Authorization", "Bearer "+token).
		Post(fmt.Sprintf("/api/admin/package-demos/feedbacks/%d/done", id), strings.NewReader(`{}`))
	require.NoError(t, err)
	content, err = resp.Content()
	require.NoError(t, err)
	var done struct {
		Code int `json:"code"`
		Data struct {
			Feedback struct {
				Status uint8 `json:"status"`
			} `json:"feedback"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(content), &done), content)
	require.Equal(t, 200, done.Code, content)
	assert.EqualValues(t, 1, done.Data.Feedback.Status)

	resp, err = testCase.Http(t).
		WithHeader("Authorization", "Bearer "+token).
		Get("/api/admin/package-demos/feedbacks?page=1&page_size=10")
	require.NoError(t, err)
	content, err = resp.Content()
	require.NoError(t, err)
	assert.Contains(t, content, `"code":200`)

	resp, err = testCase.Http(t).
		WithHeader("Authorization", "Bearer "+token).
		Delete(fmt.Sprintf("/api/admin/package-demos/feedbacks/%d", id), nil)
	require.NoError(t, err)
	content, err = resp.Content()
	require.NoError(t, err)
	assert.Contains(t, content, `"code":200`)
}
