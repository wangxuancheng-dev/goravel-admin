package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/goravel/framework/facades"

	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/tenancy"
)

const (
	TenantHealthOK      = "ok"
	TenantHealthWarn    = "warn"
	TenantHealthFail    = "fail"
	TenantHealthUnknown = "unknown"

	HealthIssuePingFail      = "ping_fail"
	HealthIssueSchemaBehind  = "schema_behind"
	HealthIssueSchemaFailed  = "schema_failed"
	HealthIssueProvisionFail = "provision_failed"
	HealthIssueMigrateFail   = "migrate_failed"
	HealthIssueQuotaHigh     = "storage_quota_high"
	HealthIssueQuotaFull     = "storage_quota_full"
	HealthIssueDomainFail    = "domain_verify_failed"
)

// TenantHealthInspectResult is one tenant inspect outcome.
type TenantHealthInspectResult struct {
	TenantID uint     `json:"tenant_id"`
	Code     string   `json:"code"`
	Status   string   `json:"status"`
	Issues   []string `json:"issues"`
	PingOK   bool     `json:"ping_ok"`
	PingMs   int64    `json:"ping_ms"`
}

// TenantHealthInspectReport summarizes a full run.
type TenantHealthInspectReport struct {
	Checked int                         `json:"checked"`
	OK      int                         `json:"ok"`
	Warn    int                         `json:"warn"`
	Fail    int                         `json:"fail"`
	Results []TenantHealthInspectResult `json:"results"`
}

// RunTenantHealthInspect pings and evaluates schema/quota/provision for active tenants.
func RunTenantHealthInspect(limit int, alert bool) (*TenantHealthInspectReport, error) {
	if !tenancy.Enabled() {
		return &TenantHealthInspectReport{Results: []TenantHealthInspectResult{}}, nil
	}
	if limit <= 0 {
		limit = 200
	}
	if limit > 500 {
		limit = 500
	}
	var tenants []models.Tenant
	// Prefer never-checked / oldest-checked so large fleets rotate past Limit.
	if err := appfacades.PlatformOrmQuery(nil).Model(&models.Tenant{}).
		Where("status", models.TenantStatusActive).
		Order("health_checked_at IS NULL DESC").
		Order("health_checked_at ASC").
		Order("id ASC").
		Limit(limit).
		Find(&tenants); err != nil {
		return nil, err
	}
	report := &TenantHealthInspectReport{Results: make([]TenantHealthInspectResult, 0, len(tenants))}
	ids := make([]uint, 0, len(tenants))
	for i := range tenants {
		ids = append(ids, tenants[i].ID)
	}
	domainMeta := LoadTenantDomainListMeta(ids)
	conn := NewTenantConnectionService()
	expected := ExpectedSchemaMigrationCount()

	failCodes := make([]string, 0)
	warnCodes := make([]string, 0)

	for i := range tenants {
		t := &tenants[i]
		res := inspectOneTenant(conn, t, expected, domainMeta[t.ID])
		now := time.Now()
		_, _ = appfacades.PlatformOrmQuery(nil).Model(t).Update(map[string]any{
			"health_status":     res.Status,
			"health_checked_at": now,
			"health_issues":     encodeHealthIssues(res.Issues),
			"last_ping_ok":      res.PingOK,
			"last_ping_ms":      res.PingMs,
		})
		report.Checked++
		switch res.Status {
		case TenantHealthOK:
			report.OK++
		case TenantHealthWarn:
			report.Warn++
			warnCodes = append(warnCodes, t.Code)
		case TenantHealthFail:
			report.Fail++
			failCodes = append(failCodes, t.Code)
		}
		report.Results = append(report.Results, res)
	}

	if alert && (len(failCodes) > 0 || len(warnCodes) > 0) {
		AlertTenantHealthIfNeeded(failCodes, warnCodes, report)
	}
	return report, nil
}

func inspectOneTenant(
	conn *TenantConnectionService,
	t *models.Tenant,
	expected int64,
	domain TenantDomainListMeta,
) TenantHealthInspectResult {
	res := TenantHealthInspectResult{
		TenantID: t.ID,
		Code:     t.Code,
		Status:   TenantHealthOK,
		Issues:   []string{},
	}

	if t.ProvisionStatus == models.TenantProvisionFailed {
		res.Issues = append(res.Issues, HealthIssueProvisionFail)
	}
	if strings.TrimSpace(t.LastMigrateError) != "" && t.ProvisionStatus != models.TenantProvisionReady {
		res.Issues = append(res.Issues, HealthIssueMigrateFail)
	}
	if t.LastOp == models.TenantOpMigrate && t.LastOpStatus == models.TenantOpStatusFailed {
		res.Issues = append(res.Issues, HealthIssueMigrateFail)
	}

	schemaSt := ResolveTenantSchemaStatus(t, expected)
	switch schemaSt {
	case TenantSchemaBehind:
		res.Issues = append(res.Issues, HealthIssueSchemaBehind)
	case TenantSchemaFailed:
		res.Issues = append(res.Issues, HealthIssueSchemaFailed)
	}

	if domain.Status == TenantDomainStatusVerifyFailed {
		res.Issues = append(res.Issues, HealthIssueDomainFail)
	}

	if t.IsProvisionReady() {
		ping := conn.BuildPingDetail(t)
		if ping == nil || !ping.OK {
			res.PingOK = false
			res.Issues = append(res.Issues, HealthIssuePingFail)
			if ping != nil {
				res.PingMs = ping.LatencyMs
			}
		} else {
			res.PingOK = true
			res.PingMs = ping.LatencyMs
		}
	}

	if t.StorageLimitBytes > 0 {
		quota := BuildTenantQuotaSnapshot(t)
		if quota.StorageUsedBytes >= t.StorageLimitBytes {
			res.Issues = append(res.Issues, HealthIssueQuotaFull)
		} else if float64(quota.StorageUsedBytes) >= float64(t.StorageLimitBytes)*0.9 {
			res.Issues = append(res.Issues, HealthIssueQuotaHigh)
		}
	}

	res.Status = classifyHealth(res.Issues)
	return res
}

func classifyHealth(issues []string) string {
	if len(issues) == 0 {
		return TenantHealthOK
	}
	fail := map[string]struct{}{
		HealthIssuePingFail:      {},
		HealthIssueSchemaFailed:  {},
		HealthIssueProvisionFail: {},
		HealthIssueMigrateFail:   {},
		HealthIssueQuotaFull:     {},
	}
	for _, issue := range issues {
		if _, ok := fail[issue]; ok {
			return TenantHealthFail
		}
	}
	return TenantHealthWarn
}

// AlertTenantHealthIfNeeded posts webhook (WeCom-compatible JSON) when health degrades.
func AlertTenantHealthIfNeeded(failCodes, warnCodes []string, report *TenantHealthInspectReport) {
	url := ResolveTenantOpsAlertWebhookURL()
	mailTo := strings.TrimSpace(facades.Config().GetString("health.tenant_health_alert_mail", ""))
	if (url == "" && mailTo == "") || report == nil {
		return
	}
	key := fmt.Sprintf("health:tenant_health_alert:%s", time.Now().Format("2006-01-02-15"))
	if facades.Cache().GetString(key, "") != "" {
		return
	}
	_ = facades.Cache().Put(key, "1", time.Hour)

	msg := fmt.Sprintf("tenant health: fail=%d warn=%d checked=%d", report.Fail, report.Warn, report.Checked)
	if len(failCodes) > 0 {
		msg += "; fail=[" + joinHealthCodes(failCodes, 8) + "]"
	}
	if len(warnCodes) > 0 {
		msg += "; warn=[" + joinHealthCodes(warnCodes, 8) + "]"
	}

	if url != "" {
		payload := map[string]any{
			"msgtype": "text",
			"text": map[string]any{
				"content": msg,
			},
			"event":      "tenant_health_inspect",
			"fail":       report.Fail,
			"warn":       report.Warn,
			"checked":    report.Checked,
			"fail_codes": failCodes,
			"warn_codes": warnCodes,
			"timestamp":  time.Now().Unix(),
			"app":        facades.Config().GetString("app.name", ""),
			"env":        facades.Config().GetString("app.env", ""),
		}
		go deliverPlatformWebhookAlert(models.PlatformAlertChannelWebhook, "tenant_health_inspect", nil, "", "", url, payload)
	}

	if mailTo != "" {
		subject := "[tenant-health] " + facades.Config().GetString("app.name", "goravel-admin")
		go deliverPlatformMailAlert("tenant_health_inspect", mailTo, subject, msg)
	}
}

func joinHealthCodes(codes []string, max int) string {
	if len(codes) > max {
		codes = codes[:max]
	}
	return strings.Join(codes, ",")
}

// TenantHealthAlertConfigStatus reports health alert channels without secrets.
func TenantHealthAlertConfigStatus() map[string]any {
	webhook := ResolveTenantOpsAlertWebhookURL()
	mailTo := strings.TrimSpace(facades.Config().GetString("health.tenant_health_alert_mail", ""))
	return map[string]any{
		"webhook_configured": webhook != "",
		"mail_configured":    mailTo != "",
		"mail_masked":        maskEmail(mailTo),
	}
}

func maskEmail(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	at := strings.Index(raw, "@")
	if at <= 1 {
		return "***"
	}
	return raw[:1] + "***" + raw[at:]
}
