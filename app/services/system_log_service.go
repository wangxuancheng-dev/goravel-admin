package services

import (
	"context"
	"encoding/json"
	"sort"
	"sync"
	"time"

	"github.com/goravel/framework/contracts/http"

	"goravel/app/constants"
	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/http/helpers"
	"goravel/app/models"
	"goravel/app/tenancy"
	"goravel/app/utils/traceid"
)

type SystemLogService interface {
	// GetByID 根据ID获取系统日志
	GetByID(id uint) (*models.SystemLog, error)
	// GetList 获取系统日志列表
	GetList(filters SystemLogFilters, page, pageSize int) ([]models.SystemLog, int64, error)
	Delete(id uint) error
	BatchDelete(ids []uint) error
	Clean(days int) error
	GetModuleOptions() []string
	// RecordHTTP 记录系统日志（HTTP context）
	RecordHTTP(ctx http.Context, level, module, message string, attributes map[string]any) error
	// Record 记录系统日志（标准 context）
	Record(ctx context.Context, level, module, message string, attributes map[string]any) error
}

// SystemLogFilters 系统日志查询过滤器
type SystemLogFilters struct {
	Level     string
	Module    string
	TraceID   string
	Message   string
	StartTime string
	EndTime   string
	OrderBy   string
}

func BuildSystemLogFiltersFromHTTP(ctx http.Context) SystemLogFilters {
	return SystemLogFilters{
		Level:     ctx.Request().Query("level", ""),
		Module:    ctx.Request().Query("module", ""),
		TraceID:   ctx.Request().Query("trace_id", ""),
		Message:   ctx.Request().Query("message", ""),
		StartTime: helpers.GetTimeInputOrQueryParam(ctx, "start_time"),
		EndTime:   helpers.GetTimeInputOrQueryParam(ctx, "end_time"),
		OrderBy:   ctx.Request().Query("order_by", ""),
	}
}

type SystemLogServiceImpl struct {
	ctx context.Context
}

var (
	systemLogsTraceIDCache sync.Map // connection -> bool
)

func (s *SystemLogServiceImpl) hasTraceIDColumn() bool {
	return s.hasTraceIDColumnCtx(s.ctx)
}

func (s *SystemLogServiceImpl) hasTraceIDColumnCtx(ctx context.Context) bool {
	key := appfacades.SchemaConnectionKeyFrom(ctx)
	if tenancy.Enabled() && !tenancy.Bound(ctx) {
		key = "platform:" + appfacades.PlatformConnectionName()
	}
	if v, ok := systemLogsTraceIDCache.Load(key); ok {
		return v.(bool)
	}
	has := appfacades.SchemaHasTable(ctx, "system_logs") && appfacades.SchemaHasColumn(ctx, "system_logs", "trace_id")
	systemLogsTraceIDCache.Store(key, has)
	return has
}

func NewSystemLogService(ctx context.Context) SystemLogService {
	return &SystemLogServiceImpl{ctx: ctx}
}

// GetByID 根据ID获取系统日志
func (s *SystemLogServiceImpl) GetByID(id uint) (*models.SystemLog, error) {
	var log models.SystemLog
	if err := appfacades.SystemLogOrmQuery(s.ctx).Where("id", id).FirstOrFail(&log); err != nil {
		return nil, apperrors.ErrLogNotFound.WithError(err)
	}
	return &log, nil
}

// GetList 获取系统日志列表
func (s *SystemLogServiceImpl) GetList(filters SystemLogFilters, page, pageSize int) ([]models.SystemLog, int64, error) {
	query := appfacades.SystemLogOrmQuery(s.ctx).Model(&models.SystemLog{})

	// 应用筛选条件
	if filters.Level != "" {
		query = query.Where("level = ?", filters.Level)
	}
	if filters.Module != "" {
		query = query.Where("module LIKE ?", "%"+filters.Module+"%")
	}
	if filters.TraceID != "" {
		if s.hasTraceIDColumn() {
			query = query.Where("trace_id LIKE ?", "%"+filters.TraceID+"%")
		}
	}
	if filters.Message != "" {
		query = query.Where("message LIKE ?", "%"+filters.Message+"%")
	}
	if filters.StartTime != "" {
		query = query.Where("created_at >= ?", filters.StartTime)
	}
	if filters.EndTime != "" {
		query = query.Where("created_at <= ?", filters.EndTime)
	}

	// 应用排序
	orderBy := filters.OrderBy
	if orderBy == "" {
		orderBy = "id:desc"
	}
	query = helpers.ApplySort(query, orderBy, "id:desc")

	// 分页查询
	var logs []models.SystemLog
	var total int64
	if err := query.Paginate(page, pageSize, &logs, &total); err != nil {
		return nil, 0, apperrors.ErrQueryFailed.WithError(err)
	}

	return logs, total, nil
}

func (s *SystemLogServiceImpl) Delete(id uint) error {
	log, err := s.GetByID(id)
	if err != nil {
		return err
	}
	if _, err := appfacades.SystemLogOrmQuery(s.ctx).Delete(log); err != nil {
		return apperrors.ErrDeleteFailed.WithError(err)
	}
	return nil
}

func (s *SystemLogServiceImpl) BatchDelete(ids []uint) error {
	if len(ids) == 0 {
		return apperrors.ErrIDsRequired
	}
	idsAny := helpers.ConvertUintSliceToAny(ids)
	if _, err := appfacades.SystemLogOrmQuery(s.ctx).WhereIn("id", idsAny).Delete(&models.SystemLog{}); err != nil {
		return apperrors.ErrDeleteFailed.WithError(err)
	}
	return nil
}

func (s *SystemLogServiceImpl) Clean(days int) error {
	if days <= 0 {
		days = constants.DefaultCleanLogDays
	}
	cutoffTime := time.Now().AddDate(0, 0, -days)
	if _, err := appfacades.SystemLogOrmQuery(s.ctx).Model(&models.SystemLog{}).Where("created_at < ?", cutoffTime).Delete(&models.SystemLog{}); err != nil {
		return apperrors.ErrDeleteFailed.WithError(err)
	}
	return nil
}

func (s *SystemLogServiceImpl) GetModuleOptions() []string {
	var modules []string
	_ = appfacades.SystemLogOrmQuery(s.ctx).Model(&models.SystemLog{}).
		Select("DISTINCT module").
		Where("module IS NOT NULL AND module != ''").
		Order("module ASC").
		Pluck("module", &modules)

	moduleSet := make(map[string]struct{}, len(modules))
	uniqueModules := make([]string, 0, len(modules))
	for _, module := range modules {
		if module == "" {
			continue
		}
		if _, exists := moduleSet[module]; exists {
			continue
		}
		moduleSet[module] = struct{}{}
		uniqueModules = append(uniqueModules, module)
	}
	sort.Strings(uniqueModules)
	return uniqueModules
}

// RecordHTTP 记录系统日志（HTTP context）
func (s *SystemLogServiceImpl) RecordHTTP(ctx http.Context, level, module, message string, attributes map[string]any) (err error) {
	// Must not panic: used from Route Recover where a secondary panic becomes http.Server crash.
	defer func() {
		if rec := recover(); rec != nil {
			err = nil
		}
	}()

	if appfacades.Orm() == nil {
		return nil
	}

	var contextJSON string
	if len(attributes) > 0 {
		if data, marshalErr := json.Marshal(attributes); marshalErr == nil {
			contextJSON = string(data)
		}
	}

	traceID := traceid.FromHTTPContext(ctx)
	if traceID == "" {
		traceID = traceid.EnsureHTTPContext(ctx, "")
	}

	ip, userAgent := "", ""
	if ctx != nil && ctx.Request() != nil {
		ip = ctx.Request().Ip()
		userAgent = ctx.Request().Header("User-Agent", "")
	}

	payload := map[string]any{
		"level":      level,
		"module":     module,
		"message":    message,
		"context":    contextJSON,
		"ip":         ip,
		"user_agent": userAgent,
		"created_at": time.Now(),
		"updated_at": time.Now(),
	}

	q := appfacades.SystemLogOrmQuery(ctx)
	if q == nil {
		return nil
	}
	if s.hasTraceIDColumnCtx(ctx) {
		payload["trace_id"] = traceID
	}
	return q.Table("system_logs").Create(payload)
}

// Record 记录系统日志（标准 context）
func (s *SystemLogServiceImpl) Record(ctx context.Context, level, module, message string, attributes map[string]any) (err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = nil
		}
	}()

	if appfacades.Orm() == nil {
		return nil
	}

	var contextJSON string
	if len(attributes) > 0 {
		if data, marshalErr := json.Marshal(attributes); marshalErr == nil {
			contextJSON = string(data)
		}
	}

	traceID := traceid.FromContext(ctx)
	if traceID == "" {
		var newCtx context.Context
		newCtx, traceID = traceid.EnsureContext(ctx)
		ctx = newCtx
	}

	payload := map[string]any{
		"level":      level,
		"module":     module,
		"message":    message,
		"context":    contextJSON,
		"created_at": time.Now(),
		"updated_at": time.Now(),
	}

	q := appfacades.SystemLogOrmQuery(ctx)
	if q == nil {
		return nil
	}
	if s.hasTraceIDColumnCtx(ctx) {
		payload["trace_id"] = traceID
	}
	return q.Table("system_logs").Create(payload)
}
