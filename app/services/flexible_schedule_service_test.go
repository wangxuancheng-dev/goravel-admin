package services

import (
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"goravel/app/models"
)

func TestDailyAtToCron(t *testing.T) {
	assert.Equal(t, "0 20 * * *", dailyAtToCron("20:00"))
	assert.Equal(t, "30 2 * * *", dailyAtToCron("02:30"))
	assert.Equal(t, "0 20 * * *", dailyAtToCron("bad"))
}

func TestValidateCronExpr(t *testing.T) {
	require.NoError(t, ValidateCronExpr("*/5 * * * *"))
	require.NoError(t, ValidateCronExpr("0 20 * * *"))
	require.Error(t, ValidateCronExpr(""))
	require.Error(t, ValidateCronExpr("not a cron"))
}

func TestCronMatchesMinute(t *testing.T) {
	// 2026-09-21 20:00:30 UTC should match "0 20 * * *"
	now := time.Date(2026, 9, 21, 20, 0, 30, 0, time.UTC)
	slot, ok := CronMatchesMinute("0 20 * * *", "UTC", now)
	assert.True(t, ok)
	assert.Equal(t, "2026-09-21T20:00", slot)

	_, ok = CronMatchesMinute("0 20 * * *", "UTC", now.Add(time.Minute))
	assert.False(t, ok)

	// Asia/Shanghai 04:00 = UTC 20:00 previous day... pick local noon
	loc, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)
	local := time.Date(2026, 9, 22, 9, 15, 0, 0, loc)
	slot, ok = CronMatchesMinute("15 9 * * *", "Asia/Shanghai", local.UTC())
	assert.True(t, ok)
	assert.Equal(t, "2026-09-22T09:15", slot)
}

func TestPreviewNextRunsFormat(t *testing.T) {
	from := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	runs, err := PreviewNextRuns("0 20 * * *", "UTC", from, 2)
	require.NoError(t, err)
	require.Len(t, runs, 2)
	assert.Equal(t, "2026-09-22 20:00:00", runs[0])
	assert.Equal(t, "2026-09-23 20:00:00", runs[1])
}

func TestTruncateFlexibleOutput(t *testing.T) {
	assert.Equal(t, "abc", truncateFlexibleOutput("abc", 10))
	s := "你好世界"
	out := truncateFlexibleOutput(s, 7) // mid-rune cut should land on rune boundary
	assert.True(t, utf8.ValidString(out))
	assert.LessOrEqual(t, len(out), 7)
}

func TestNormalizeFlexiblePayload(t *testing.T) {
	got, err := normalizeFlexiblePayload("")
	require.NoError(t, err)
	assert.Equal(t, "{}", got)

	got, err = normalizeFlexiblePayload(`{"keep":2}`)
	require.NoError(t, err)
	assert.Equal(t, `{"keep":2}`, got)

	_, err = normalizeFlexiblePayload(`[]`)
	require.Error(t, err)
}

func TestFlexiblePayloadMap(t *testing.T) {
	row := &models.FlexibleSchedule{Payload: `{"a":1}`}
	m := FlexiblePayloadMap(row)
	assert.EqualValues(t, 1, m["a"])
	assert.Empty(t, FlexiblePayloadMap(nil))
}

