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

	"goravel/app/tenancyctx"
	"goravel/app/utils"
)

// DispatchAuditWebhook POSTs an audit event when AUDIT_WEBHOOK_URL (or tenant
// configs audit.webhook_url) is set. Never blocks the caller.
func DispatchAuditWebhook(ctx context.Context, event string, payload map[string]any) {
	url := strings.TrimSpace(facades.Config().GetString("audit.webhook_url", ""))
	if ctx != nil {
		if v := strings.TrimSpace(utils.GetConfigValue(ctx, "audit", "webhook_url", "")); v != "" {
			url = v
		}
	}
	if url == "" {
		return
	}
	body := map[string]any{
		"event":     event,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"payload":   payload,
	}
	if code, ok := tenancyctx.CodeFrom(ctx); ok && code != "" {
		body["tenant_code"] = code
	}
	if id, ok := tenancyctx.IDFrom(ctx); ok && id > 0 {
		body["tenant_id"] = id
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				facades.Log().Errorf("audit webhook panic: %v", r)
			}
		}()
		if err := postAuditJSON(url, body); err != nil {
			facades.Log().Warningf("audit webhook failed event=%s err=%v", event, err)
		}
	}()
}

func postAuditJSON(url string, payload map[string]any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook status %d", resp.StatusCode)
	}
	return nil
}
