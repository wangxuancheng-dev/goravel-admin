package health

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/goravel/framework/facades"
)

const (
	readyAlertDebounceKey = "health:ready_alert_debounce"
	readyAlertLastStatus  = "health:ready_last_status"
	readyAlertDebounceTTL = 5 * time.Minute
)

// MaybeAlertNotReady POSTs JSON to READY_ALERT_WEBHOOK_URL when readiness fails,
// preferring transition-to-not-ready and debouncing repeats for 5 minutes.
func MaybeAlertNotReady(report Report) {
	prev := facades.Cache().GetString(readyAlertLastStatus, "")
	_ = facades.Cache().Put(readyAlertLastStatus, report.Status, 24*time.Hour)

	if report.Status != "not_ready" {
		return
	}

	url := strings.TrimSpace(facades.Config().GetString("health.ready_alert_webhook_url", ""))
	if url == "" {
		return
	}

	if facades.Cache().GetString(readyAlertDebounceKey, "") != "" {
		return
	}
	// Alert on first not-ready or after debounce TTL while still down.
	_ = prev
	_ = facades.Cache().Put(readyAlertDebounceKey, "1", readyAlertDebounceTTL)
	go postReadyAlert(url, report)
}

func postReadyAlert(url string, report Report) {
	payload := map[string]any{
		"event":     "ready_failure",
		"status":    report.Status,
		"timestamp": report.Timestamp,
		"checks":    report.Checks,
		"app":       facades.Config().GetString("app.name", ""),
		"env":       facades.Config().GetString("app.env", ""),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		facades.Log().Warningf("ready alert webhook build request failed: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		facades.Log().Warningf("ready alert webhook post failed: %v", err)
		return
	}
	_ = resp.Body.Close()
}
