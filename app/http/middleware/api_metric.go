package middleware

import (
	"context"
	appfacades "goravel/app/facades"
	"regexp"
	"strings"
	"time"

	"github.com/goravel/framework/contracts/http"

	appmetrics "goravel/app/metrics"
	"goravel/app/models"
	"goravel/app/tenancyctx"
	"goravel/app/utils/logger"
	"goravel/app/utils/traceid"
)

var (
	digitsPattern = regexp.MustCompile(`/\d+`)
	uuidPattern   = regexp.MustCompile(`/[0-9a-fA-F]{8}-[0-9a-fA-F-]{27,36}`)
)

// ApiMetric collects per-request API metrics for performance observability.
func ApiMetric() http.Middleware {
	return newMiddleware("api_metric", func(ctx http.Context) {
		startAt := time.Now()
		method := strings.ToUpper(strings.TrimSpace(ctx.Request().Method()))
		path := strings.TrimSpace(ctx.Request().Path())

		ctx.Request().Next()

		if !shouldCollectAPIMetric(method, path) {
			return
		}

		durationMS := float64(time.Since(startAt).Milliseconds())
		if durationMS < 0 {
			durationMS = 0
		}
		traceID := traceid.FromHTTPContext(ctx)
		statusCode := extractResponseStatusCode(ctx)
		routeTemplate := resolveRouteTemplate(ctx, path)
		occurredAt := time.Now().Format("2006-01-02 15:04:05")

		metric := models.ApiEndpointMetric{
			TraceID:       traceID,
			Method:        method,
			RouteTemplate: routeTemplate,
			StatusCode:    statusCode,
			DurationMS:    durationMS,
			OccurredAt:    occurredAt,
		}

		appmetrics.ObserveHTTP(method, routeTemplate, statusCode, time.Since(startAt))

		persistCtx := tenancyctx.Detach(ctx)
		go func(data models.ApiEndpointMetric, dbCtx context.Context) {
			// 请求返回后 HTTP ctx 会被取消；保留租户连接键并落库。
			if err := appfacades.OrmQuery(dbCtx).Create(&data); err != nil {
				logger.ErrorfContext(dbCtx, "persist api endpoint metric failed: %v", err)
			}
		}(metric, persistCtx)
	})
}

func shouldCollectAPIMetric(method, path string) bool {
	if method == "" || path == "" {
		return false
	}
	if !strings.HasPrefix(path, "/api/admin/") {
		return false
	}
	if strings.HasPrefix(path, "/api/admin/observability/") {
		return false
	}
	if strings.HasPrefix(path, "/api/admin/heartbeat") {
		return false
	}
	if strings.HasPrefix(path, "/api/admin/login/captcha") {
		return false
	}
	return true
}

func resolveRouteTemplate(ctx http.Context, path string) string {
	if slug, ok := ctx.Value("permission_slug").(string); ok && strings.TrimSpace(slug) != "" {
		return slug
	}
	normalized := uuidPattern.ReplaceAllString(path, "/{uuid}")
	normalized = digitsPattern.ReplaceAllString(normalized, "/{id}")
	if normalized == "" {
		return "unknown"
	}
	return normalized
}

func extractResponseStatusCode(ctx http.Context) int {
	type statusCarrier interface {
		Status() int
	}
	if res, ok := any(ctx.Response()).(statusCarrier); ok {
		if status := res.Status(); status > 0 {
			return status
		}
	}
	return 200
}
