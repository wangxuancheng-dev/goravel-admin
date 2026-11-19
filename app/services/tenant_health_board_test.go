package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"goravel/app/models"
)

func TestBuildTenantHealthBoard(t *testing.T) {
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	fresh := now.Add(-30 * time.Minute)
	old := now.Add(-3 * time.Hour)

	tenants := []models.Tenant{
		{Code: "ok1", HealthStatus: "ok", HealthIssues: "[]", HealthCheckedAt: &fresh},
		{Code: "warn1", HealthStatus: "warn", HealthIssues: `["schema_behind"]`, HealthCheckedAt: &fresh},
		{Code: "fail1", HealthStatus: "fail", HealthIssues: `["ping_fail","schema_behind","ping_fail"]`, HealthCheckedAt: &old},
		{Code: "new1", HealthStatus: "", HealthIssues: ""},
	}
	tenants[0].ID, tenants[1].ID, tenants[2].ID, tenants[3].ID = 1, 2, 3, 4

	board := BuildTenantHealthBoard(tenants, now)

	assert.Equal(t, int64(3), board.Checked)
	assert.Equal(t, int64(1), board.NeverChecked)
	assert.Equal(t, int64(2), board.Stale) // fail1 (old) + new1 (never)
	assert.Equal(t, fresh, *board.LastCheckedAt)
	assert.Equal(t, int64(1), board.ByStatus["ok"])
	assert.Equal(t, int64(1), board.ByStatus["warn"])
	assert.Equal(t, int64(1), board.ByStatus["fail"])
	assert.Equal(t, int64(1), board.ByStatus["unknown"])
	assert.Equal(t, int64(2), board.ByIssue["schema_behind"])
	assert.Equal(t, int64(1), board.ByIssue["ping_fail"]) // deduped per tenant
	assert.Equal(t, int64(2), board.Abnormal)

	if assert.Len(t, board.Items, 2) {
		assert.Equal(t, "fail1", board.Items[0].Code) // fail sorts first
		assert.Equal(t, []string{"ping_fail", "schema_behind"}, board.Items[0].Issues)
		assert.True(t, board.Items[0].Stale)
		assert.Equal(t, "warn1", board.Items[1].Code)
		assert.False(t, board.Items[1].Stale)
	}
}
