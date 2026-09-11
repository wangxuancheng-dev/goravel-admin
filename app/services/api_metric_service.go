package services

import (
	"context"
	appfacades "goravel/app/facades"
	"sort"
	"time"

	"goravel/app/models"
)

type ApiPerformanceItem struct {
	Method        string  `json:"method"`
	RouteTemplate string  `json:"route_template"`
	Count         int64   `json:"count"`
	ErrorCount    int64   `json:"error_count"`
	ErrorRate     float64 `json:"error_rate"`
	AvgDurationMS float64 `json:"avg_duration_ms"`
	MaxDurationMS float64 `json:"max_duration_ms"`
	P95DurationMS float64 `json:"p95_duration_ms"`
	P99DurationMS float64 `json:"p99_duration_ms"`
	QPS           float64 `json:"qps"`
}

type ApiPerformanceTraceItem struct {
	TraceID       string  `json:"trace_id"`
	Method        string  `json:"method"`
	RouteTemplate string  `json:"route_template"`
	StatusCode    int     `json:"status_code"`
	DurationMS    float64 `json:"duration_ms"`
	OccurredAt    string  `json:"occurred_at"`
}

type ApiPerformanceOverview struct {
	WindowHours int                  `json:"window_hours"`
	Limit       int                  `json:"limit"`
	SlowTop     []ApiPerformanceItem `json:"slow_top"`
	ErrorTop    []ApiPerformanceItem `json:"error_top"`
	QPSTop      []ApiPerformanceItem `json:"qps_top"`
}

type ApiMetricService interface {
	GetOverview(hours, limit int) (ApiPerformanceOverview, error)
	GetRecentTraces(method, routeTemplate string, hours, limit int) ([]ApiPerformanceTraceItem, error)
}

type ApiMetricServiceImpl struct {
	ctx context.Context
}

type apiAggregateRow struct {
	Method        string  `json:"method"`
	RouteTemplate string  `json:"route_template"`
	TotalCount    int64   `json:"total_count"`
	ErrorCount    int64   `json:"error_count"`
	AvgDurationMS float64 `json:"avg_duration_ms"`
	MaxDurationMS float64 `json:"max_duration_ms"`
}

func NewApiMetricService(ctx context.Context) ApiMetricService {
	return &ApiMetricServiceImpl{ctx: ctx}
}

func (s *ApiMetricServiceImpl) GetOverview(hours, limit int) (ApiPerformanceOverview, error) {
	if hours <= 0 {
		hours = 24
	}
	if hours > 24*7 {
		hours = 24 * 7
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	overview := ApiPerformanceOverview{
		WindowHours: hours,
		Limit:       limit,
		SlowTop:     []ApiPerformanceItem{},
		ErrorTop:    []ApiPerformanceItem{},
		QPSTop:      []ApiPerformanceItem{},
	}
	if !appfacades.SchemaHasTable(s.ctx, "api_endpoint_metrics") {
		return overview, nil
	}

	cutoff := time.Now().Add(-time.Duration(hours) * time.Hour)
	var rows []apiAggregateRow
	err := appfacades.OrmQuery(s.ctx).Raw(`
SELECT
	method,
	route_template,
	COUNT(*) AS total_count,
	SUM(CASE WHEN status_code >= 500 THEN 1 ELSE 0 END) AS error_count,
	AVG(duration_ms) AS avg_duration_ms,
	MAX(duration_ms) AS max_duration_ms
FROM api_endpoint_metrics
WHERE created_at >= ?
GROUP BY method, route_template
ORDER BY total_count DESC
LIMIT 5000
`, cutoff).Scan(&rows)
	if err != nil {
		return overview, err
	}

	items := make([]ApiPerformanceItem, 0, len(rows))
	windowSeconds := float64(hours * 3600)
	for _, row := range rows {
		if row.TotalCount <= 0 {
			continue
		}
		p95, p99 := s.getEndpointPercentiles(cutoff, row.Method, row.RouteTemplate)
		errorRate := 0.0
		if row.TotalCount > 0 {
			errorRate = float64(row.ErrorCount) / float64(row.TotalCount)
		}
		items = append(items, ApiPerformanceItem{
			Method:        row.Method,
			RouteTemplate: row.RouteTemplate,
			Count:         row.TotalCount,
			ErrorCount:    row.ErrorCount,
			ErrorRate:     errorRate,
			AvgDurationMS: row.AvgDurationMS,
			MaxDurationMS: row.MaxDurationMS,
			P95DurationMS: p95,
			P99DurationMS: p99,
			QPS:           float64(row.TotalCount) / windowSeconds,
		})
	}

	slowTop := append([]ApiPerformanceItem(nil), items...)
	sort.Slice(slowTop, func(i, j int) bool {
		if slowTop[i].P95DurationMS == slowTop[j].P95DurationMS {
			return slowTop[i].AvgDurationMS > slowTop[j].AvgDurationMS
		}
		return slowTop[i].P95DurationMS > slowTop[j].P95DurationMS
	})

	errorTop := append([]ApiPerformanceItem(nil), items...)
	sort.Slice(errorTop, func(i, j int) bool {
		if errorTop[i].ErrorRate == errorTop[j].ErrorRate {
			return errorTop[i].ErrorCount > errorTop[j].ErrorCount
		}
		return errorTop[i].ErrorRate > errorTop[j].ErrorRate
	})

	qpsTop := append([]ApiPerformanceItem(nil), items...)
	sort.Slice(qpsTop, func(i, j int) bool {
		if qpsTop[i].QPS == qpsTop[j].QPS {
			return qpsTop[i].Count > qpsTop[j].Count
		}
		return qpsTop[i].QPS > qpsTop[j].QPS
	})

	overview.SlowTop = trimApiItems(slowTop, limit)
	overview.ErrorTop = trimApiItems(errorTop, limit)
	overview.QPSTop = trimApiItems(qpsTop, limit)
	return overview, nil
}

func (s *ApiMetricServiceImpl) GetRecentTraces(method, routeTemplate string, hours, limit int) ([]ApiPerformanceTraceItem, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}
	if hours <= 0 {
		hours = 24
	}
	if hours > 24*7 {
		hours = 24 * 7
	}
	if !appfacades.SchemaHasTable(s.ctx, "api_endpoint_metrics") {
		return []ApiPerformanceTraceItem{}, nil
	}

	cutoff := time.Now().Add(-time.Duration(hours) * time.Hour)
	var rows []models.ApiEndpointMetric
	query := appfacades.OrmQuery(s.ctx).
		Model(&models.ApiEndpointMetric{}).
		Where("created_at >= ?", cutoff).
		Order("id desc").
		Limit(limit)
	if method != "" {
		query = query.Where("method", method)
	}
	if routeTemplate != "" {
		query = query.Where("route_template", routeTemplate)
	}
	if err := query.Get(&rows); err != nil {
		return nil, err
	}

	list := make([]ApiPerformanceTraceItem, 0, len(rows))
	for _, row := range rows {
		list = append(list, ApiPerformanceTraceItem{
			TraceID:       row.TraceID,
			Method:        row.Method,
			RouteTemplate: row.RouteTemplate,
			StatusCode:    row.StatusCode,
			DurationMS:    row.DurationMS,
			OccurredAt:    row.OccurredAt,
		})
	}
	return list, nil
}

func (s *ApiMetricServiceImpl) getEndpointPercentiles(cutoff time.Time, method, routeTemplate string) (float64, float64) {
	var rows []models.ApiEndpointMetric
	err := appfacades.OrmQuery(s.ctx).
		Model(&models.ApiEndpointMetric{}).
		Select("duration_ms").
		Where("created_at >= ?", cutoff).
		Where("method", method).
		Where("route_template", routeTemplate).
		Order("id desc").
		Limit(5000).
		Get(&rows)
	if err != nil || len(rows) == 0 {
		return 0, 0
	}

	values := make([]float64, 0, len(rows))
	for _, item := range rows {
		values = append(values, item.DurationMS)
	}
	sort.Float64s(values)
	return percentileFloat(values, 0.95), percentileFloat(values, 0.99)
}

func percentileFloat(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}
	if p <= 0 {
		return values[0]
	}
	if p >= 1 {
		return values[len(values)-1]
	}
	pos := int(float64(len(values)-1) * p)
	if pos < 0 {
		pos = 0
	}
	if pos >= len(values) {
		pos = len(values) - 1
	}
	return values[pos]
}

func trimApiItems(items []ApiPerformanceItem, limit int) []ApiPerformanceItem {
	if len(items) <= limit {
		return items
	}
	return items[:limit]
}
