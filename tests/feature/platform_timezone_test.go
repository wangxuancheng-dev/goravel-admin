package feature_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/tests"
)

// The platform console sends X-Timezone; every *_at in /api/platform responses is
// shown in that zone (stored UTC), not only created_at/updated_at.
func TestPlatformResponseTimesFollowXTimezone(t *testing.T) {
	withTenancyDriver(t, "database")
	token := loginPlatformSmoke(t)

	event := fmt.Sprintf("tz_test_%d", time.Now().UnixNano()%1000000)
	delivered := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	row := models.PlatformAlertDelivery{Channel: models.PlatformAlertChannelWebhook, Event: event, Status: models.PlatformAlertStatusSuccess, Attempt: 1, DeliveredAt: &delivered}
	require.NoError(t, appfacades.PlatformQuery().Create(&row))
	t.Cleanup(func() {
		_, _ = appfacades.PlatformQuery().Where("id", row.ID).ForceDelete(&models.PlatformAlertDelivery{})
	})

	fetch := func(tz string) map[string]any {
		req := (&tests.TestCase{}).Http(t).WithHeader("Authorization", "Bearer "+token)
		if tz != "" {
			req = req.WithHeader("X-Timezone", tz)
		}
		resp, err := req.Get("/api/platform/alert-deliveries?event=" + event)
		require.NoError(t, err)
		content, err := resp.Content()
		require.NoError(t, err)
		var payload struct {
			Data struct {
				List []map[string]any `json:"list"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal([]byte(content), &payload), content)
		require.Len(t, payload.Data.List, 1, content)
		return payload.Data.List[0]
	}

	utc := fetch("")
	sh := fetch("Asia/Shanghai")

	assert.Equal(t, "2026-01-02 11:04:05", sh["delivered_at"], "RFC3339 *_at converted to the request zone")
	assert.Contains(t, fmt.Sprint(utc["delivered_at"]), "03:04:05", "no X-Timezone: stored UTC")

	utcCreated, err := time.Parse("2006-01-02 15:04:05", fmt.Sprint(utc["created_at"]))
	require.NoError(t, err, "created_at: %v", utc["created_at"])
	shCreated, err := time.Parse("2006-01-02 15:04:05", fmt.Sprint(sh["created_at"]))
	require.NoError(t, err, "created_at: %v", sh["created_at"])
	assert.Equal(t, 8*time.Hour, shCreated.Sub(utcCreated))
}
