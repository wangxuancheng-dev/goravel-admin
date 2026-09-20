package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/goravel/framework/contracts/mail"
	"github.com/goravel/framework/facades"

	"goravel/app/models"
)

const tenantOpsAlertDebounceTTL = 30 * time.Minute

// ResolveTenantOpsAlertWebhookURL returns TENANT_OPS_ALERT_WEBHOOK_URL,
// else QUEUE_ALERT_WEBHOOK_URL, else READY_ALERT_WEBHOOK_URL.
func ResolveTenantOpsAlertWebhookURL() string {
	url := strings.TrimSpace(facades.Config().GetString("health.tenant_ops_alert_webhook_url", ""))
	if url != "" {
		return url
	}
	url = strings.TrimSpace(facades.Config().GetString("health.queue_alert_webhook_url", ""))
	if url != "" {
		return url
	}
	return strings.TrimSpace(facades.Config().GetString("health.ready_alert_webhook_url", ""))
}

// TenantOpsAlertConfigStatus reports webhook configuration without exposing secrets.
func TenantOpsAlertConfigStatus() map[string]any {
	opsURL := strings.TrimSpace(facades.Config().GetString("health.tenant_ops_alert_webhook_url", ""))
	queueURL := strings.TrimSpace(facades.Config().GetString("health.queue_alert_webhook_url", ""))
	readyURL := strings.TrimSpace(facades.Config().GetString("health.ready_alert_webhook_url", ""))
	url := ResolveTenantOpsAlertWebhookURL()
	configured := url != ""
	source := ""
	switch {
	case opsURL != "":
		source = "TENANT_OPS_ALERT_WEBHOOK_URL"
	case queueURL != "":
		source = "QUEUE_ALERT_WEBHOOK_URL"
	case readyURL != "":
		source = "READY_ALERT_WEBHOOK_URL"
	}
	masked := ""
	if configured {
		masked = maskOpsWebhookURL(url)
	}
	return map[string]any{
		"configured": configured,
		"source":     source,
		"url_masked": masked,
	}
}

func maskOpsWebhookURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if len(raw) <= 12 {
		return "***"
	}
	return raw[:8] + "…" + raw[len(raw)-4:]
}

// AlertTenantOpFailedIfNeeded POSTs when a platform tenant op fails (debounced per tenant+op).
func AlertTenantOpFailedIfNeeded(tenant *models.Tenant, op, message string) {
	url := ResolveTenantOpsAlertWebhookURL()
	if url == "" || tenant == nil {
		return
	}
	op = strings.TrimSpace(op)
	if op == "" {
		op = "unknown"
	}
	key := fmt.Sprintf("health:tenant_ops_alert:%d:%s", tenant.ID, op)
	if facades.Cache().GetString(key, "") != "" {
		return
	}
	_ = facades.Cache().Put(key, "1", tenantOpsAlertDebounceTTL)

	msg := strings.TrimSpace(message)
	if len(msg) > 500 {
		msg = msg[:500]
	}
	payload := map[string]any{
		"event":                  "tenant_op_failed",
		"tenant_id":              tenant.ID,
		"tenant_code":            tenant.Code,
		"op":                     op,
		"message":                msg,
		"provision_status":       tenant.ProvisionStatus,
		"schema_migration_count": tenant.SchemaMigrationCount,
		"timestamp":              time.Now().Unix(),
		"app":                    facades.Config().GetString("app.name", ""),
		"env":                    facades.Config().GetString("app.env", ""),
	}
	tid := tenant.ID
	go deliverPlatformWebhookAlert(models.PlatformAlertChannelWebhook, "tenant_op_failed", &tid, tenant.Code, op, url, payload)
}

func deliverPlatformWebhookAlert(channel, event string, tenantID *uint, tenantCode, op, url string, payload map[string]any) {
	body, _ := json.Marshal(payload)
	row := &models.PlatformAlertDelivery{
		Channel:      channel,
		Event:        event,
		TenantID:     tenantID,
		TenantCode:   tenantCode,
		Op:           op,
		Status:       models.PlatformAlertStatusPending,
		TargetMasked: maskOpsWebhookURL(url),
		Payload:      string(body),
		Attempt:      1,
	}
	RecordPlatformAlertDelivery(row)
	httpStatus, err := postTenantOpsWebhookWithStatus(url, payload)
	if err != nil {
		facades.Log().Warningf("tenant ops alert webhook post failed: %v", err)
		FinishPlatformAlertDelivery(row.ID, models.PlatformAlertStatusFailed, httpStatus, err.Error())
		return
	}
	FinishPlatformAlertDelivery(row.ID, models.PlatformAlertStatusSuccess, httpStatus, "")
}

func postTenantOpsWebhook(url string, payload map[string]any) error {
	_, err := postTenantOpsWebhookWithStatus(url, payload)
	return err
}

func postTenantOpsWebhookWithStatus(url string, payload map[string]any) (int, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}
	reqCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 300 {
		return resp.StatusCode, fmt.Errorf("webhook status %d", resp.StatusCode)
	}
	return resp.StatusCode, nil
}

// deliverPlatformMailAlert records and sends health alert mail.
func deliverPlatformMailAlert(event, to, subject, body string) {
	row := &models.PlatformAlertDelivery{
		Channel:      models.PlatformAlertChannelMail,
		Event:        event,
		Status:       models.PlatformAlertStatusPending,
		TargetMasked: maskEmail(to),
		Payload:      body,
		Attempt:      1,
	}
	RecordPlatformAlertDelivery(row)
	err := facades.Mail().To([]string{to}).Subject(subject).Content(mail.Content{Html: "<pre>" + body + "</pre>"}).Send()
	if err != nil {
		facades.Log().Warningf("tenant health alert mail failed: %v", err)
		FinishPlatformAlertDelivery(row.ID, models.PlatformAlertStatusFailed, 0, err.Error())
		return
	}
	FinishPlatformAlertDelivery(row.ID, models.PlatformAlertStatusSuccess, 0, "")
}
