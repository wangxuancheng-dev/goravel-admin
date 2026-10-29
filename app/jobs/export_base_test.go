package jobs

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseArgs(t *testing.T) {
	want := ExportArgs{
		ExportID: 7,
		AdminID:  3,
		TenantID: 9,
		Filters:  map[string]any{"status": "paid"},
		Type:     "orders",
		Language: "en",
		Timezone: "Asia/Shanghai",
	}

	t.Run("struct", func(t *testing.T) {
		got, err := ParseArgs(want)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("json string", func(t *testing.T) {
		got, err := ParseArgs(`{"export_id":7,"admin_id":3,"tenant_id":9,"filters":{"status":"paid"},"type":"orders","language":"en","timezone":"Asia/Shanghai"}`)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("map from queue payload", func(t *testing.T) {
		got, err := ParseArgs(map[string]any{
			"export_id": float64(7),
			"admin_id":  float64(3),
			"tenant_id": float64(9),
			"filters":   map[string]any{"status": "paid"},
			"type":      "orders",
			"language":  "en",
			"timezone":  "Asia/Shanghai",
		})
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	errCases := map[string][]any{
		"no args":           nil,
		"unsupported type":  {42},
		"missing export_id": {ExportArgs{AdminID: 1}},
		"zero id in map":    {map[string]any{"admin_id": float64(1)}},
	}
	for name, args := range errCases {
		t.Run(name, func(t *testing.T) {
			_, err := ParseArgs(args...)
			assert.Error(t, err)
		})
	}
}

func TestFormatTimeWithTimezone(t *testing.T) {
	ts := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	assert.Equal(t, "", FormatTimeWithTimezone(time.Time{}, "UTC"))
	assert.Equal(t, "2026-01-02 03:04:05", FormatTimeWithTimezone(ts, ""))
	assert.Equal(t, "2026-01-02 11:04:05", FormatTimeWithTimezone(ts, "Asia/Shanghai"))
	// Unknown zones fall back to the time's own location instead of failing the export.
	assert.Equal(t, "2026-01-02 03:04:05", FormatTimeWithTimezone(ts, "Not/AZone"))
}

func TestExportExecutionLockTTL(t *testing.T) {
	def := defaultExportExecutionLockTTLSeconds * time.Second
	cases := map[string]time.Duration{
		"":     def,
		"abc":  def,
		"0":    def,
		"-5":   def,
		"1800": 1800 * time.Second,
	}
	for raw, want := range cases {
		t.Run("env="+raw, func(t *testing.T) {
			t.Setenv("QUEUE_EXPORT_EXECUTION_LOCK_TTL_SECONDS", raw)
			assert.Equal(t, want, getExportExecutionLockTTL())
		})
	}
}
