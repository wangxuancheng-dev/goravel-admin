package health

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/goravel/framework/facades"

	"goravel/app/services"
)

const (
	queueAlertDebounceKey = "health:queue_alert_backlog_debounce"
	queueAlertDebounceTTL = time.Hour
)

// QueueBacklogSnapshot is a cheap pending/failed summary for ops alerts.
type QueueBacklogSnapshot struct {
	Connection string           `json:"connection"`
	Kind       string           `json:"kind"`
	Pending    int64            `json:"pending"`
	Failed     int64            `json:"failed"`
	Threshold  int64            `json:"threshold"`
	ByQueue    map[string]int64 `json:"by_queue,omitempty"`
}

// ResolveQueueAlertWebhookURL returns QUEUE_ALERT_WEBHOOK_URL, else READY_ALERT_WEBHOOK_URL.
func ResolveQueueAlertWebhookURL() string {
	url := strings.TrimSpace(facades.Config().GetString("health.queue_alert_webhook_url", ""))
	if url != "" {
		return url
	}
	return strings.TrimSpace(facades.Config().GetString("health.ready_alert_webhook_url", ""))
}

// QueueAlertBacklogThreshold returns configured pending threshold (default 100).
func QueueAlertBacklogThreshold() int64 {
	n := facades.Config().GetInt("health.queue_alert_backlog_threshold", 100)
	if n <= 0 {
		return 100
	}
	return int64(n)
}

// CollectRedisQueueBacklog sums pending/failed for the default Redis queue connection.
// Returns nil, nil when the default connection is not a Redis driver (caller should skip).
func CollectRedisQueueBacklog(ctx context.Context) (*QueueBacklogSnapshot, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	reader := services.NewQueueStatsReader(ctx)
	connection := facades.Config().GetString("queue.default", "sync")
	if !reader.IsRedisDriver(connection) {
		return nil, nil
	}

	redisConn := reader.GetRedisConnectionName(connection)
	byQueue, err := reader.GetRedisStatsByQueue(redisConn, connection)
	if err != nil {
		// Fall back to default logical queue only (cheap single-key path).
		def := facades.Config().GetString(fmt.Sprintf("queue.connections.%s.queue", connection), "default")
		single, e2 := reader.GetRedisQueueStats(redisConn, connection, def)
		if e2 != nil {
			return nil, err
		}
		byQueue = map[string]*services.RedisQueueStatsInfo{def: single}
	}

	snap := &QueueBacklogSnapshot{
		Connection: connection,
		Kind:       "redis_list",
		Threshold:  QueueAlertBacklogThreshold(),
		ByQueue:    make(map[string]int64, len(byQueue)),
	}
	if reader.IsRedisStreamDriver(connection) {
		snap.Kind = "redis_stream"
	}
	for name, st := range byQueue {
		if st == nil {
			continue
		}
		snap.Pending += st.Pending
		snap.Failed += st.Failed
		snap.ByQueue[name] = st.Pending
	}
	return snap, nil
}

// AlertQueueBacklogIfNeeded POSTs a webhook when Redis pending exceeds threshold.
// Debounced via cache (1 hour). Returns whether an alert was sent (or would send).
func AlertQueueBacklogIfNeeded(ctx context.Context) (sent bool, snap *QueueBacklogSnapshot, err error) {
	url := ResolveQueueAlertWebhookURL()
	if url == "" {
		return false, nil, nil
	}

	snap, err = CollectRedisQueueBacklog(ctx)
	if err != nil {
		return false, nil, err
	}
	if snap == nil {
		return false, nil, nil
	}
	if snap.Pending <= snap.Threshold {
		return false, snap, nil
	}

	if facades.Cache().GetString(queueAlertDebounceKey, "") != "" {
		return false, snap, nil
	}
	_ = facades.Cache().Put(queueAlertDebounceKey, "1", queueAlertDebounceTTL)
	go postQueueBacklogAlert(url, snap)
	return true, snap, nil
}

func postQueueBacklogAlert(url string, snap *QueueBacklogSnapshot) {
	payload := map[string]any{
		"event":      "queue_backlog",
		"connection": snap.Connection,
		"kind":       snap.Kind,
		"pending":    snap.Pending,
		"failed":     snap.Failed,
		"threshold":  snap.Threshold,
		"by_queue":   snap.ByQueue,
		"timestamp":  time.Now().Unix(),
		"app":        facades.Config().GetString("app.name", ""),
		"env":        facades.Config().GetString("app.env", ""),
	}
	if err := PostJSONWebhook(url, payload); err != nil {
		facades.Log().Warningf("queue backlog alert webhook post failed: %v", err)
	}
}

// PostJSONWebhook POSTs a JSON payload to url (5s timeout). Returns non-nil on transport/build errors.
func PostJSONWebhook(url string, payload map[string]any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	reqCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook status %d", resp.StatusCode)
	}
	return nil
}

// QueueAlertConfigStatus reports whether webhook env is set (never exposes the URL secret fully).
func QueueAlertConfigStatus() map[string]any {
	url := ResolveQueueAlertWebhookURL()
	configured := url != ""
	source := ""
	if strings.TrimSpace(facades.Config().GetString("health.queue_alert_webhook_url", "")) != "" {
		source = "QUEUE_ALERT_WEBHOOK_URL"
	} else if configured {
		source = "READY_ALERT_WEBHOOK_URL"
	}
	masked := ""
	if configured {
		masked = maskWebhookURL(url)
	}
	return map[string]any{
		"configured": configured,
		"source":     source,
		"url_masked": masked,
		"threshold":  QueueAlertBacklogThreshold(),
	}
}

func maskWebhookURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if len(raw) <= 12 {
		return "***"
	}
	return raw[:8] + "…" + raw[len(raw)-4:]
}

// SendTestQueueAlert posts a test event to the configured webhook (no debounce).
func SendTestQueueAlert(ctx context.Context) (map[string]any, error) {
	url := ResolveQueueAlertWebhookURL()
	if url == "" {
		return nil, fmt.Errorf("queue_alert_webhook_not_configured")
	}
	snap, err := CollectRedisQueueBacklog(ctx)
	payload := map[string]any{
		"event":     "queue_alert_test",
		"test":      true,
		"timestamp": time.Now().Unix(),
		"app":       facades.Config().GetString("app.name", ""),
		"env":       facades.Config().GetString("app.env", ""),
		"message":   "manual test from observability queue alert",
	}
	if err == nil && snap != nil {
		payload["connection"] = snap.Connection
		payload["kind"] = snap.Kind
		payload["pending"] = snap.Pending
		payload["failed"] = snap.Failed
		payload["threshold"] = snap.Threshold
	}
	if err := PostJSONWebhook(url, payload); err != nil {
		return nil, err
	}
	return QueueAlertConfigStatus(), nil
}
