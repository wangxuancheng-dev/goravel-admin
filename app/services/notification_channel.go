package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/goravel/framework/facades"

	appfacades "goravel/app/facades"
	"goravel/app/models"
)

// EnqueueEmailFn dispatches a send_email job. Wired from providers to avoid services↔jobs import cycle.
var EnqueueEmailFn = func(to, subject, content string) error {
	return nil
}

// NotificationMailSubject builds the email subject for a notification.
func NotificationMailSubject(n *models.Notification) string {
	if n == nil {
		return "Notification"
	}
	title := strings.TrimSpace(n.Title)
	if title == "" {
		title = "Notification"
	}
	return fmt.Sprintf("[%s] %s", notificationTypeLabel(n.Type), title)
}

// NotificationMailBody builds a simple HTML email body.
func NotificationMailBody(n *models.Notification) string {
	if n == nil {
		return ""
	}
	return fmt.Sprintf(`<h2>%s</h2><div>%s</div><p style="color:#888;font-size:12px;">type: %s</p>`,
		escapeBasic(n.Title), n.Content, escapeBasic(n.Type))
}

func notificationTypeLabel(t string) string {
	switch t {
	case "announcement":
		return "Announcement"
	case "message":
		return "Message"
	default:
		if t == "" {
			return "Notice"
		}
		return t
	}
}

func escapeBasic(s string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
	)
	return replacer.Replace(s)
}

// DispatchNotificationChannels sends optional mail / webhook after create (best-effort).
func DispatchNotificationChannels(ctx context.Context, notifications ...*models.Notification) {
	if len(notifications) == 0 {
		return
	}
	mailEnabled := facades.Config().GetBool("notification.mail_enabled", false)
	webhookEnabled := facades.Config().GetBool("notification.webhook_enabled", false)
	webhookURL := strings.TrimSpace(facades.Config().GetString("notification.webhook_url", ""))

	if !mailEnabled && !(webhookEnabled && webhookURL != "") {
		return
	}

	for _, n := range notifications {
		if n == nil {
			continue
		}
		if mailEnabled && n.ReceiverID != nil && *n.ReceiverID > 0 {
			dispatchNotificationMail(ctx, n)
		}
		if webhookEnabled && webhookURL != "" {
			dispatchNotificationWebhook(webhookURL, n)
		}
	}
}

func dispatchNotificationMail(ctx context.Context, n *models.Notification) {
	var admin models.Admin
	if err := appfacades.OrmQuery(ctx).Where("id", *n.ReceiverID).First(&admin); err != nil {
		return
	}
	email := strings.TrimSpace(admin.Email)
	if email == "" {
		return
	}

	if err := EnqueueEmailFn(email, NotificationMailSubject(n), NotificationMailBody(n)); err != nil {
		facades.Log().Warningf("notification mail enqueue failed: %v", err)
	}
}

func dispatchNotificationWebhook(url string, n *models.Notification) {
	payload := map[string]any{
		"id":          n.ID,
		"title":       n.Title,
		"content":     n.Content,
		"type":        n.Type,
		"sender_id":   n.SenderID,
		"receiver_id": n.ReceiverID,
		"created_at":  time.Now().UTC().Format(time.RFC3339),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return
	}

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		facades.Log().Warningf("notification webhook build request failed: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		facades.Log().Warningf("notification webhook post failed: %v", err)
		return
	}
	_ = resp.Body.Close()
}
