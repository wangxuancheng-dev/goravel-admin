package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/goravel/framework/facades"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/tenancyctx"
)

// TenantAdminSupportItem is a read-only tenant admin row for platform support.
type TenantAdminSupportItem struct {
	ID           uint   `json:"id"`
	Username     string `json:"username"`
	Nickname     string `json:"nickname"`
	Status       uint8  `json:"status"`
	Is2FABound   bool   `json:"is_2fa_bound"`
	MustChangePw uint8  `json:"must_change_password"`
}

// TenantAuditSummary is a read-only cross-tenant audit snapshot.
type TenantAuditSummary struct {
	LoginSuccess24h   int64                    `json:"login_success_24h"`
	LoginFailed24h    int64                    `json:"login_failed_24h"`
	OperationCount24h int64                    `json:"operation_count_24h"`
	RecentLogins      []map[string]any         `json:"recent_logins"`
	RecentOperations  []map[string]any         `json:"recent_operations"`
	Error             string                   `json:"error,omitempty"`
}

func bindTenantSupportCtx(tenant *models.Tenant) (context.Context, error) {
	if err := NewTenantConnectionService().EnsureRegistered(tenant); err != nil {
		return nil, err
	}
	return tenancyctx.WithTenant(context.Background(), tenant.ID, tenant.ConnectionName, tenant.Code), nil
}

// ListTenantAdmins returns tenant-db admin accounts (support use).
func ListTenantAdmins(tenant *models.Tenant, limit int) ([]TenantAdminSupportItem, error) {
	if tenant == nil {
		return nil, apperrors.ErrInvalidArgument
	}
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	ctx, err := bindTenantSupportCtx(tenant)
	if err != nil {
		return nil, err
	}
	out := make([]TenantAdminSupportItem, 0)
	if !appfacades.SchemaHasTable(ctx, "admins") {
		return out, nil
	}
	type row struct {
		ID                 uint
		Username           string
		Nickname           string
		Status             uint8
		GoogleSecret       string
		MustChangePassword uint8
	}
	var rows []row
	q := appfacades.OrmQuery(ctx).Table("admins").
		Select("id", "username", "nickname", "status", "google_secret", "must_change_password").
		Order("id asc").Limit(limit)
	if err := q.Find(&rows); err != nil {
		return nil, err
	}
	for _, r := range rows {
		out = append(out, TenantAdminSupportItem{
			ID:           r.ID,
			Username:     r.Username,
			Nickname:     r.Nickname,
			Status:       r.Status,
			Is2FABound:   strings.TrimSpace(r.GoogleSecret) != "",
			MustChangePw: r.MustChangePassword,
		})
	}
	return out, nil
}

// ResetTenantAdminPassword sets a new password and forces change-on-login.
func ResetTenantAdminPassword(tenant *models.Tenant, adminID uint, newPassword string) error {
	if tenant == nil || adminID == 0 {
		return apperrors.ErrInvalidArgument
	}
	ctx, err := bindTenantSupportCtx(tenant)
	if err != nil {
		return err
	}
	if err := ValidatePasswordPolicyCtx(ctx, newPassword); err != nil {
		return err
	}
	hashed, err := facades.Hash().Make(newPassword)
	if err != nil {
		return apperrors.ErrPasswordEncryptFailed.WithError(err)
	}
	var admin models.Admin
	if err := appfacades.OrmQuery(ctx).Where("id", adminID).FirstOrFail(&admin); err != nil {
		return apperrors.ErrAdminNotFound.WithError(err)
	}
	admin.Password = hashed
	admin.MustChangePassword = 1
	return appfacades.OrmQuery(ctx).Save(&admin)
}

// UnlockTenantAdminLogin clears login lockout for a tenant admin username.
func UnlockTenantAdminLogin(tenant *models.Tenant, username string) error {
	if tenant == nil {
		return apperrors.ErrInvalidArgument
	}
	username = strings.TrimSpace(username)
	if username == "" {
		return apperrors.ErrInvalidArgument
	}
	ctx, err := bindTenantSupportCtx(tenant)
	if err != nil {
		return err
	}
	NewLoginLockoutService(ctx).UnlockUsername(username)
	return nil
}

// ResetTenantAdmin2FA clears Google Authenticator binding.
func ResetTenantAdmin2FA(tenant *models.Tenant, adminID uint) error {
	if tenant == nil || adminID == 0 {
		return apperrors.ErrInvalidArgument
	}
	ctx, err := bindTenantSupportCtx(tenant)
	if err != nil {
		return err
	}
	var admin models.Admin
	if err := appfacades.OrmQuery(ctx).Where("id", adminID).FirstOrFail(&admin); err != nil {
		return apperrors.ErrAdminNotFound.WithError(err)
	}
	if strings.TrimSpace(admin.GoogleSecret) == "" {
		return apperrors.ErrGoogleAuthenticatorNotBound
	}
	_, err = appfacades.OrmQuery(ctx).Model(&models.Admin{}).Where("id", adminID).Update("google_secret", nil)
	return err
}

// BuildTenantAuditSummary reads recent login/operation logs from the tenant DB.
func BuildTenantAuditSummary(tenant *models.Tenant, recentLimit int) *TenantAuditSummary {
	out := &TenantAuditSummary{
		RecentLogins:     []map[string]any{},
		RecentOperations: []map[string]any{},
	}
	if tenant == nil {
		out.Error = "tenant is nil"
		return out
	}
	if recentLimit <= 0 || recentLimit > 50 {
		recentLimit = 10
	}
	ctx, err := bindTenantSupportCtx(tenant)
	if err != nil {
		out.Error = err.Error()
		return out
	}
	since := time.Now().Add(-24 * time.Hour)
	q := appfacades.OrmQuery(ctx)
	if appfacades.SchemaHasTable(ctx, "login_logs") {
		ok, _ := q.Table("login_logs").
			Where("status", 1).Where("created_at", ">=", since).Count()
		out.LoginSuccess24h = ok
		fail, _ := q.Table("login_logs").
			Where("status", 0).Where("created_at", ">=", since).Count()
		out.LoginFailed24h = fail

		var logins []models.LoginLog
		_ = q.Table("login_logs").
			Order("id desc").Limit(recentLimit).Find(&logins)
		for _, row := range logins {
			out.RecentLogins = append(out.RecentLogins, map[string]any{
				"id":         row.ID,
				"username":   row.Username,
				"ip":         row.IP,
				"status":     row.Status,
				"message":    row.Message,
				"created_at": row.CreatedAt,
			})
		}
	}
	if appfacades.SchemaHasTable(ctx, "operation_logs") {
		n, _ := q.Table("operation_logs").
			Where("created_at", ">=", since).Count()
		out.OperationCount24h = n

		var ops []models.OperationLog
		_ = q.Model(&models.OperationLog{}).
			Order("id desc").Limit(recentLimit).Find(&ops)
		adminNames := map[uint]string{}
		for _, row := range ops {
			username := adminNames[row.AdminID]
			if username == "" && row.AdminID > 0 {
				var a models.Admin
				if err := q.Select("id", "username").Where("id", row.AdminID).First(&a); err == nil {
					username = a.Username
					adminNames[row.AdminID] = username
				}
			}
			out.RecentOperations = append(out.RecentOperations, map[string]any{
				"id":         row.ID,
				"username":   username,
				"title":      row.Title,
				"method":     row.Method,
				"path":       row.Path,
				"ip":         row.IP,
				"status":     row.Status,
				"created_at": row.CreatedAt,
			})
		}
	}
	return out
}

// IPInAllowlist reports whether clientIP matches comma-separated IPs/CIDRs.
// Empty allowlist means allow all.
func IPInAllowlist(clientIP, allowlist string) bool {
	allowlist = strings.TrimSpace(allowlist)
	if allowlist == "" {
		return true
	}
	ip := net.ParseIP(strings.TrimSpace(clientIP))
	if ip == nil {
		return false
	}
	for _, part := range strings.Split(allowlist, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Contains(part, "/") {
			_, network, err := net.ParseCIDR(part)
			if err == nil && network.Contains(ip) {
				return true
			}
			continue
		}
		if parsed := net.ParseIP(part); parsed != nil && parsed.Equal(ip) {
			return true
		}
	}
	return false
}

// RecordPlatformAlertDelivery inserts a delivery row (best-effort).
func RecordPlatformAlertDelivery(row *models.PlatformAlertDelivery) {
	if row == nil {
		return
	}
	if row.Attempt == 0 {
		row.Attempt = 1
	}
	if row.Status == "" {
		row.Status = models.PlatformAlertStatusPending
	}
	_ = appfacades.PlatformOrmQuery(nil).Create(row)
}

// FinishPlatformAlertDelivery updates status after send.
func FinishPlatformAlertDelivery(id uint, status string, httpStatus int, errMsg string) {
	if id == 0 {
		return
	}
	updates := map[string]any{
		"status":        status,
		"http_status":   httpStatus,
		"error_message": truncateAlertErr(errMsg, 1000),
	}
	if status == models.PlatformAlertStatusSuccess {
		now := time.Now()
		updates["delivered_at"] = now
	}
	_, _ = appfacades.PlatformOrmQuery(nil).Model(&models.PlatformAlertDelivery{}).Where("id", id).Update(updates)
}

func truncateAlertErr(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// PlatformAlertDeliveryFilters for list.
type PlatformAlertDeliveryFilters struct {
	Event      string
	Status     string
	Channel    string
	TenantCode string
}

// ListPlatformAlertDeliveries pages alert delivery rows.
func ListPlatformAlertDeliveries(filters PlatformAlertDeliveryFilters, page, pageSize int) ([]models.PlatformAlertDelivery, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	query := appfacades.PlatformOrmQuery(nil).Model(&models.PlatformAlertDelivery{})
	if v := strings.TrimSpace(filters.Event); v != "" {
		query = query.Where("event", v)
	}
	if v := strings.TrimSpace(filters.Status); v != "" {
		query = query.Where("status", v)
	}
	if v := strings.TrimSpace(filters.Channel); v != "" {
		query = query.Where("channel", v)
	}
	if v := strings.TrimSpace(filters.TenantCode); v != "" {
		query = query.Where("tenant_code", v)
	}
	var total int64
	total, err := query.Count()
	if err != nil {
		return nil, 0, err
	}
	var list []models.PlatformAlertDelivery
	if err := query.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list); err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// PlatformAlertDeliveryToJSON maps a delivery row.
func PlatformAlertDeliveryToJSON(row *models.PlatformAlertDelivery) map[string]any {
	if row == nil {
		return nil
	}
	return map[string]any{
		"id":            row.ID,
		"channel":       row.Channel,
		"event":         row.Event,
		"tenant_id":     row.TenantID,
		"tenant_code":   row.TenantCode,
		"op":            row.Op,
		"status":        row.Status,
		"http_status":   row.HTTPStatus,
		"target_masked": row.TargetMasked,
		"error_message": row.ErrorMessage,
		"attempt":       row.Attempt,
		"delivered_at":  row.DeliveredAt,
		"created_at":    row.CreatedAt,
		"updated_at":    row.UpdatedAt,
	}
}

// RetryPlatformAlertDelivery re-sends a stored webhook payload.
func RetryPlatformAlertDelivery(id uint) (*models.PlatformAlertDelivery, error) {
	var row models.PlatformAlertDelivery
	if err := appfacades.PlatformOrmQuery(nil).Where("id", id).FirstOrFail(&row); err != nil {
		return nil, apperrors.ErrRecordNotFound.WithError(err)
	}
	if row.Channel != models.PlatformAlertChannelWebhook {
		return nil, apperrors.ErrInvalidArgument.WithMessage("only webhook retries are supported")
	}
	url := ResolveTenantOpsAlertWebhookURL()
	if url == "" {
		return nil, apperrors.ErrInvalidArgument.WithMessage("webhook not configured")
	}
	var payload map[string]any
	if strings.TrimSpace(row.Payload) != "" {
		if err := json.Unmarshal([]byte(row.Payload), &payload); err != nil {
			return nil, apperrors.ErrInvalidArgument.WithError(err)
		}
	} else {
		payload = map[string]any{
			"event":       row.Event,
			"tenant_code": row.TenantCode,
			"op":          row.Op,
			"retry_of":    row.ID,
			"timestamp":   time.Now().Unix(),
		}
	}
	payload["retry_of"] = row.ID
	payload["timestamp"] = time.Now().Unix()

	httpStatus, err := postTenantOpsWebhookWithStatus(url, payload)
	row.Attempt++
	row.HTTPStatus = uint(httpStatus)
	row.TargetMasked = maskOpsWebhookURL(url)
	now := time.Now()
	if err != nil {
		row.Status = models.PlatformAlertStatusFailed
		row.ErrorMessage = truncateAlertErr(err.Error(), 1000)
	} else {
		row.Status = models.PlatformAlertStatusSuccess
		row.ErrorMessage = ""
		row.DeliveredAt = &now
	}
	if saveErr := appfacades.PlatformOrmQuery(nil).Save(&row); saveErr != nil {
		return nil, saveErr
	}
	if err != nil {
		return &row, fmt.Errorf("%w", err)
	}
	return &row, nil
}
