package services

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/goravel/framework/support/carbon"

	appfacades "goravel/app/facades"
	"goravel/app/models"
)

type ObservabilityService interface {
	TraceAggregate(traceID string, slowSQL any) (map[string]any, error)
	CollectAuditEvents(traceID, keyword string, adminID int, startTime, endTime string) ([]AuditEvent, error)
	LoadTraceOperations(traceID string) ([]models.OperationLog, error)
	LoadTraceSystemLogs(traceID string) ([]models.SystemLog, error)
}

type AuditEvent struct {
	Time       string         `json:"time"`
	SortAt     int64          `json:"-"`
	Type       string         `json:"type"`
	TraceID    string         `json:"trace_id"`
	Title      string         `json:"title,omitempty"`
	Method     string         `json:"method,omitempty"`
	Path       string         `json:"path,omitempty"`
	AdminID    uint           `json:"admin_id,omitempty"`
	AdminName  string         `json:"admin_name,omitempty"`
	Status     uint8          `json:"status,omitempty"`
	Level      string         `json:"level,omitempty"`
	Module     string         `json:"module,omitempty"`
	Message    string         `json:"message,omitempty"`
	DurationMS int            `json:"duration_ms,omitempty"`
	Context    any            `json:"context,omitempty"`
}

type ObservabilityServiceImpl struct {
	ctx context.Context
}

func NewObservabilityService(ctx context.Context) ObservabilityService {
	return &ObservabilityServiceImpl{ctx: ctx}
}

func (s *ObservabilityServiceImpl) LoadTraceOperations(traceID string) ([]models.OperationLog, error) {
	var operations []models.OperationLog
	err := appfacades.OrmQuery(s.ctx).
		Model(&models.OperationLog{}).
		Where("trace_id", traceID).
		With("Admin").
		Order("id asc").
		Limit(50).
		Get(&operations)
	return operations, err
}

func (s *ObservabilityServiceImpl) LoadTraceSystemLogs(traceID string) ([]models.SystemLog, error) {
	var systemLogs []models.SystemLog
	err := appfacades.OrmQuery(s.ctx).
		Model(&models.SystemLog{}).
		Where("trace_id", traceID).
		Order("id asc").
		Limit(200).
		Get(&systemLogs)
	return systemLogs, err
}

func (s *ObservabilityServiceImpl) TraceAggregate(traceID string, slowSQL any) (map[string]any, error) {
	operations, err := s.LoadTraceOperations(traceID)
	if err != nil {
		return nil, err
	}
	systemLogs, err := s.LoadTraceSystemLogs(traceID)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"trace_id":    traceID,
		"request":     firstOperationRequest(operations),
		"operations":  operations,
		"exceptions":  filterExceptionLogs(systemLogs),
		"system_logs": systemLogs,
		"slow_sql":    slowSQL,
	}, nil
}

func (s *ObservabilityServiceImpl) CollectAuditEvents(traceID, keyword string, adminID int, startTime, endTime string) ([]AuditEvent, error) {
	opQuery := appfacades.OrmQuery(s.ctx).Model(&models.OperationLog{}).With("Admin").Order("id desc").Limit(500)
	if traceID != "" {
		opQuery = opQuery.Where("trace_id = ?", traceID)
	}
	if adminID > 0 {
		opQuery = opQuery.Where("admin_id = ?", adminID)
	}
	if keyword != "" {
		opQuery = opQuery.Where("path LIKE ? OR title LIKE ? OR request LIKE ? OR error_msg LIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	if startTime != "" {
		opQuery = opQuery.Where("created_at >= ?", startTime)
	}
	if endTime != "" {
		opQuery = opQuery.Where("created_at <= ?", endTime)
	}
	var opLogs []models.OperationLog
	if err := opQuery.Get(&opLogs); err != nil {
		return nil, err
	}

	sysQuery := appfacades.OrmQuery(s.ctx).Model(&models.SystemLog{}).Order("id desc").Limit(500)
	if traceID != "" {
		sysQuery = sysQuery.Where("trace_id = ?", traceID)
	}
	if keyword != "" {
		sysQuery = sysQuery.Where("message LIKE ? OR module LIKE ? OR context LIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	if startTime != "" {
		sysQuery = sysQuery.Where("created_at >= ?", startTime)
	}
	if endTime != "" {
		sysQuery = sysQuery.Where("created_at <= ?", endTime)
	}
	var sysLogs []models.SystemLog
	if err := sysQuery.Get(&sysLogs); err != nil {
		return nil, err
	}

	events := make([]AuditEvent, 0, len(opLogs)+len(sysLogs))
	for _, item := range opLogs {
		events = append(events, AuditEvent{
			Time:       formatCarbon(item.CreatedAt),
			SortAt:     toUnix(item.CreatedAt),
			Type:       "operation",
			TraceID:    item.TraceID,
			Title:      item.Title,
			Method:     item.Method,
			Path:       item.Path,
			AdminID:    item.AdminID,
			AdminName:  item.Admin.Username,
			Status:     item.Status,
			Message:    item.ErrorMsg,
			DurationMS: item.Duration,
			Context: map[string]any{
				"request": item.Request,
				"changes": safeJSON(item.Changes),
			},
		})
	}
	for _, item := range sysLogs {
		events = append(events, AuditEvent{
			Time:    formatCarbon(item.CreatedAt),
			SortAt:  toUnix(item.CreatedAt),
			Type:    "system",
			TraceID: item.TraceID,
			Level:   item.Level,
			Module:  item.Module,
			Message: item.Message,
			Context: safeJSON(item.Context),
		})
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].SortAt > events[j].SortAt
	})
	return events, nil
}

func firstOperationRequest(operations []models.OperationLog) map[string]any {
	if len(operations) == 0 {
		return map[string]any{}
	}
	first := operations[0]
	return map[string]any{
		"trace_id":    first.TraceID,
		"path":        first.Path,
		"method":      first.Method,
		"status":      first.Status,
		"request":     safeJSON(first.Request),
		"duration_ms": first.Duration,
		"created_at":  first.CreatedAt,
	}
}

func formatCarbon(dt *carbon.DateTime) string {
	if dt == nil {
		return ""
	}
	return dt.ToDateTimeString()
}

func toUnix(dt *carbon.DateTime) int64 {
	if dt == nil {
		return 0
	}
	parsed, err := time.Parse("2006-01-02 15:04:05", dt.ToDateTimeString())
	if err != nil {
		return 0
	}
	return parsed.Unix()
}

func filterExceptionLogs(logs []models.SystemLog) []models.SystemLog {
	items := make([]models.SystemLog, 0)
	for _, item := range logs {
		level := strings.ToLower(item.Level)
		if level == "error" || strings.Contains(strings.ToLower(item.Message), "panic") || strings.Contains(strings.ToLower(item.Message), "exception") {
			items = append(items, item)
		}
	}
	return items
}

func safeJSON(raw string) any {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var decoded any
	if err := json.Unmarshal([]byte(raw), &decoded); err == nil {
		return decoded
	}
	return raw
}
