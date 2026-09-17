package services

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/http/helpers"
	"goravel/app/models"
	"goravel/app/utils"
)

// PlatformLoginLogFilters for platform console login log list.
type PlatformLoginLogFilters struct {
	Username  string
	IP        string
	Status    string
	StartTime string
	EndTime   string
	OrderBy   string
}

// PlatformOperationLogFilters for platform console operation log list.
type PlatformOperationLogFilters struct {
	Username  string
	Method    string
	Path      string
	Status    string
	StartTime string
	EndTime   string
	OrderBy   string
}

// BuildPlatformLoginLogFiltersFromHTTP reads list query params.
func BuildPlatformLoginLogFiltersFromHTTP(ctx http.Context) PlatformLoginLogFilters {
	return PlatformLoginLogFilters{
		Username:  ctx.Request().Query("username", ""),
		IP:        ctx.Request().Query("ip", ""),
		Status:    ctx.Request().Query("status", ""),
		StartTime: helpers.GetTimeInputOrQueryParam(ctx, "start_time"),
		EndTime:   helpers.GetTimeInputOrQueryParam(ctx, "end_time"),
		OrderBy:   ctx.Request().Query("order_by", ""),
	}
}

// BuildPlatformOperationLogFiltersFromHTTP reads list query params.
func BuildPlatformOperationLogFiltersFromHTTP(ctx http.Context) PlatformOperationLogFilters {
	return PlatformOperationLogFilters{
		Username:  ctx.Request().Query("username", ""),
		Method:    ctx.Request().Query("method", ""),
		Path:      ctx.Request().Query("path", ""),
		Status:    ctx.Request().Query("status", ""),
		StartTime: helpers.GetTimeInputOrQueryParam(ctx, "start_time"),
		EndTime:   helpers.GetTimeInputOrQueryParam(ctx, "end_time"),
		OrderBy:   ctx.Request().Query("order_by", ""),
	}
}

// SanitizePlatformRequestBody returns JSON of request inputs with sensitive fields masked.
func SanitizePlatformRequestBody(ctx http.Context) string {
	inputs := make(map[string]any)
	for key, value := range ctx.Request().All() {
		if utils.IsSensitiveField(key) {
			inputs[key] = "***"
		} else {
			inputs[key] = value
		}
	}
	data, err := json.Marshal(inputs)
	if err != nil {
		return ""
	}
	return string(data)
}

// RecordPlatformLoginLog writes one landlord login audit row (best-effort).
func RecordPlatformLoginLog(ctx http.Context, adminID uint, username string, status uint8, message string) {
	ip := helpers.GetRealIP(ctx)
	row := models.PlatformLoginLog{
		AdminID:   adminID,
		Username:  username,
		IP:        ip,
		UserAgent: ctx.Request().Header("User-Agent", ""),
		Location:  "",
		Status:    status,
		Message:   message,
		Request:   SanitizePlatformRequestBody(ctx),
	}
	if err := appfacades.PlatformOrmQuery(ctx).Create(&row); err != nil {
		facades.Log().Errorf("Failed to create platform login log: %v", err)
		return
	}

	go func(logID uint, clientIP string) {
		defer func() {
			if r := recover(); r != nil {
				facades.Log().Errorf("Recovered from panic in platform login log IP location: %v", r)
			}
		}()
		bg, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		location := utils.GetIPLocation(clientIP)
		if location == "" {
			return
		}
		if _, err := appfacades.PlatformOrmQuery(bg).
			Model(&models.PlatformLoginLog{}).
			Where("id", logID).
			Update("location", location); err != nil {
			facades.Log().Errorf("Failed to update platform login log location: %v", err)
		}
	}(row.ID, ip)
}

// CreatePlatformOperationLog persists one platform write audit row.
func CreatePlatformOperationLog(row *models.PlatformOperationLog) error {
	if row == nil {
		return nil
	}
	return appfacades.PlatformOrmQuery(context.Background()).Create(row)
}

func applyPlatformLogOrder(orderBy string) (column string, direction string) {
	column = "id"
	direction = "desc"
	orderBy = strings.TrimSpace(orderBy)
	if orderBy == "" {
		return column, direction
	}
	parts := strings.Split(orderBy, ":")
	if len(parts) >= 1 && parts[0] != "" {
		switch parts[0] {
		case "id", "created_at", "status", "username", "ip", "method", "path", "duration":
			column = parts[0]
		}
	}
	if len(parts) >= 2 && strings.EqualFold(parts[1], "asc") {
		direction = "asc"
	}
	return column, direction
}

// ListPlatformLoginLogsPaged returns filtered login logs.
func ListPlatformLoginLogsPaged(filters PlatformLoginLogFilters, page, pageSize int) ([]models.PlatformLoginLog, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	query := appfacades.PlatformOrmQuery(nil).Model(&models.PlatformLoginLog{})
	if filters.Username != "" {
		query = query.Where("username LIKE ?", "%"+filters.Username+"%")
	}
	if filters.IP != "" {
		query = query.Where("ip LIKE ?", "%"+filters.IP+"%")
	}
	if filters.Status != "" {
		query = query.Where("status", filters.Status)
	}
	if filters.StartTime != "" {
		query = query.Where("created_at >= ?", filters.StartTime)
	}
	if filters.EndTime != "" {
		query = query.Where("created_at <= ?", filters.EndTime)
	}
	total, err := query.Count()
	if err != nil {
		return nil, 0, err
	}
	col, dir := applyPlatformLogOrder(filters.OrderBy)
	var rows []models.PlatformLoginLog
	err = query.Order(col+" "+dir).Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows)
	return rows, total, err
}

// GetPlatformLoginLogByID returns one login log.
func GetPlatformLoginLogByID(id uint) (*models.PlatformLoginLog, error) {
	var row models.PlatformLoginLog
	if err := appfacades.PlatformOrmQuery(nil).Where("id", id).FirstOrFail(&row); err != nil {
		return nil, apperrors.ErrLogNotFound.WithError(err)
	}
	return &row, nil
}

// ListPlatformOperationLogsPaged returns filtered operation logs.
func ListPlatformOperationLogsPaged(filters PlatformOperationLogFilters, page, pageSize int) ([]models.PlatformOperationLog, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	query := appfacades.PlatformOrmQuery(nil).Model(&models.PlatformOperationLog{})
	if filters.Username != "" {
		query = query.Where("username LIKE ?", "%"+filters.Username+"%")
	}
	if filters.Method != "" {
		query = query.Where("method", filters.Method)
	}
	if filters.Path != "" {
		query = query.Where("path LIKE ?", "%"+filters.Path+"%")
	}
	if filters.Status != "" {
		query = query.Where("status", filters.Status)
	}
	if filters.StartTime != "" {
		query = query.Where("created_at >= ?", filters.StartTime)
	}
	if filters.EndTime != "" {
		query = query.Where("created_at <= ?", filters.EndTime)
	}
	total, err := query.Count()
	if err != nil {
		return nil, 0, err
	}
	col, dir := applyPlatformLogOrder(filters.OrderBy)
	var rows []models.PlatformOperationLog
	err = query.Order(col+" "+dir).Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows)
	return rows, total, err
}

// GetPlatformOperationLogByID returns one operation log.
func GetPlatformOperationLogByID(id uint) (*models.PlatformOperationLog, error) {
	var row models.PlatformOperationLog
	if err := appfacades.PlatformOrmQuery(nil).Where("id", id).FirstOrFail(&row); err != nil {
		return nil, apperrors.ErrLogNotFound.WithError(err)
	}
	return &row, nil
}

// PlatformLoginLogToJSON serializes a login log row.
func PlatformLoginLogToJSON(row *models.PlatformLoginLog) map[string]any {
	if row == nil {
		return nil
	}
	return map[string]any{
		"id":         row.ID,
		"admin_id":   row.AdminID,
		"username":   row.Username,
		"ip":         row.IP,
		"user_agent": row.UserAgent,
		"location":   row.Location,
		"status":     row.Status,
		"message":    row.Message,
		"request":    row.Request,
		"created_at": row.CreatedAt,
		"updated_at": row.UpdatedAt,
	}
}

// PlatformOperationLogToJSON serializes an operation log row.
func PlatformOperationLogToJSON(row *models.PlatformOperationLog) map[string]any {
	if row == nil {
		return nil
	}
	return map[string]any{
		"id":         row.ID,
		"admin_id":   row.AdminID,
		"username":   row.Username,
		"method":     row.Method,
		"path":       row.Path,
		"title":      row.Title,
		"ip":         row.IP,
		"user_agent": row.UserAgent,
		"request":    row.Request,
		"status":     row.Status,
		"error_msg":  row.ErrorMsg,
		"duration":   row.Duration,
		"created_at": row.CreatedAt,
		"updated_at": row.UpdatedAt,
	}
}

// PlatformOperationTitle builds a stable title for platform write audits.
func PlatformOperationTitle(method, path string) string {
	method = strings.ToUpper(strings.TrimSpace(method))
	path = strings.TrimSpace(path)
	switch {
	case method == "PUT" && path == "/api/platform/password":
		return "platform.password.update"
	case method == "POST" && path == "/api/platform/admins":
		return "platform.admin.create"
	case strings.HasSuffix(path, "/reset-password") && strings.Contains(path, "/admins/") && method == "POST":
		return "platform.admin.reset_password"
	case strings.HasPrefix(path, "/api/platform/admins/") && method == "PUT":
		return "platform.admin.update"
	case strings.HasPrefix(path, "/api/platform/admins/") && method == "DELETE":
		return "platform.admin.delete"
	case method == "POST" && path == "/api/platform/logout":
		return "platform.logout"
	case method == "POST" && path == "/api/platform/tenants":
		return "platform.tenant.create"
	case method == "POST" && path == "/api/platform/tenants/onboard":
		return "platform.tenant.onboard"
	case method == "POST" && path == "/api/platform/tenants/migrate-batch":
		return "platform.tenant.migrate_batch"
	case method == "POST" && path == "/api/platform/tenants/ops-batch":
		return "platform.tenant.ops_batch"
	case method == "POST" && path == "/api/platform/tenants/health-inspect":
		return "platform.tenant.health_inspect"
	case strings.HasSuffix(path, "/status") && method == "PUT":
		return "platform.tenant.status"
	case strings.HasSuffix(path, "/maintenance") && method == "PUT":
		return "platform.tenant.maintenance"
	case strings.HasSuffix(path, "/migrate") && method == "POST":
		return "platform.tenant.migrate"
	case strings.HasSuffix(path, "/seed") && method == "POST":
		return "platform.tenant.seed"
	case strings.HasSuffix(path, "/backup") && method == "POST":
		return "platform.tenant.backup"
	case strings.HasSuffix(path, "/restore") && method == "POST":
		return "platform.tenant.restore"
	case strings.HasSuffix(path, "/ping") && method == "POST":
		return "platform.tenant.ping"
	case strings.HasSuffix(path, "/undelete") && method == "POST":
		return "platform.tenant.undelete"
	case strings.HasSuffix(path, "/purge") && method == "POST":
		return "platform.tenant.purge"
	case strings.HasSuffix(path, "/force") && method == "DELETE":
		return "platform.tenant.force_delete"
	case strings.Contains(path, "/domains/") && strings.HasSuffix(path, "/verify") && method == "POST":
		return "platform.domain.verify"
	case strings.Contains(path, "/domains/") && strings.HasSuffix(path, "/primary") && method == "PUT":
		return "platform.domain.primary"
	case strings.Contains(path, "/domains/") && strings.HasSuffix(path, "/disable") && method == "PUT":
		return "platform.domain.disable"
	case strings.Contains(path, "/domains") && method == "POST":
		return "platform.domain.create"
	case strings.Contains(path, "/domains/") && method == "DELETE":
		return "platform.domain.delete"
	case strings.HasSuffix(path, "/backups/prune") && method == "POST":
		return "platform.backup.prune"
	case strings.HasPrefix(path, "/api/platform/tenants/") && method == "PUT":
		return "platform.tenant.update"
	case strings.HasPrefix(path, "/api/platform/tenants/") && method == "DELETE":
		return "platform.tenant.delete"
	default:
		return method + " " + path
	}
}
