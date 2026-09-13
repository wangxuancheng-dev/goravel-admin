package commands

import (
	"context"
	"fmt"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"

	"goravel/app/health"
)

type QueueAlertBacklog struct{}

func (r *QueueAlertBacklog) Signature() string {
	return "queue:alert-backlog"
}

func (r *QueueAlertBacklog) Description() string {
	return "Check Redis queue pending backlog and POST webhook when over threshold"
}

func (r *QueueAlertBacklog) Extend() command.Extend {
	return command.Extend{
		Category: "queue",
	}
}

func (r *QueueAlertBacklog) Handle(ctx console.Context) error {
	url := health.ResolveQueueAlertWebhookURL()
	threshold := health.QueueAlertBacklogThreshold()
	if url == "" {
		ctx.Info("queue backlog alert disabled (QUEUE_ALERT_WEBHOOK_URL / READY_ALERT_WEBHOOK_URL empty)")
		return nil
	}

	sent, snap, err := health.AlertQueueBacklogIfNeeded(context.Background())
	if err != nil {
		ctx.Error(fmt.Sprintf("queue backlog check failed: %v", err))
		return err
	}
	if snap == nil {
		ctx.Info("default queue connection is not Redis; skip backlog alert")
		return nil
	}

	ctx.Info(fmt.Sprintf(
		"connection=%s kind=%s pending=%d failed=%d threshold=%d",
		snap.Connection, snap.Kind, snap.Pending, snap.Failed, threshold,
	))
	if snap.Pending <= snap.Threshold {
		ctx.Info("backlog within threshold; no alert")
		return nil
	}
	if sent {
		ctx.Warning(fmt.Sprintf("backlog alert sent (pending=%d > %d)", snap.Pending, snap.Threshold))
	} else {
		ctx.Info("backlog over threshold but alert debounced (cache)")
	}
	return nil
}
