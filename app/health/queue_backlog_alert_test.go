package health

import (
	"testing"

	"github.com/goravel/framework/facades"
)

func TestQueueAlertBacklogThresholdDefault(t *testing.T) {
	prev := facades.Config().Get("health.queue_alert_backlog_threshold")
	t.Cleanup(func() {
		facades.Config().Add("health.queue_alert_backlog_threshold", prev)
	})

	facades.Config().Add("health.queue_alert_backlog_threshold", 0)
	if got := QueueAlertBacklogThreshold(); got != 100 {
		t.Fatalf("zero should fall back to 100, got %d", got)
	}

	facades.Config().Add("health.queue_alert_backlog_threshold", 250)
	if got := QueueAlertBacklogThreshold(); got != 250 {
		t.Fatalf("got %d", got)
	}
}

func TestResolveQueueAlertWebhookURLFallback(t *testing.T) {
	prevQ := facades.Config().GetString("health.queue_alert_webhook_url")
	prevR := facades.Config().GetString("health.ready_alert_webhook_url")
	t.Cleanup(func() {
		facades.Config().Add("health.queue_alert_webhook_url", prevQ)
		facades.Config().Add("health.ready_alert_webhook_url", prevR)
	})

	facades.Config().Add("health.queue_alert_webhook_url", "")
	facades.Config().Add("health.ready_alert_webhook_url", "https://ops.example/ready")
	if got := ResolveQueueAlertWebhookURL(); got != "https://ops.example/ready" {
		t.Fatalf("expected ready fallback, got %q", got)
	}

	facades.Config().Add("health.queue_alert_webhook_url", "https://ops.example/queue")
	if got := ResolveQueueAlertWebhookURL(); got != "https://ops.example/queue" {
		t.Fatalf("expected queue url, got %q", got)
	}
}
