package metrics

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/goravel/framework/facades"
)

var (
	registry = prometheus.NewRegistry()

	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "goravel_admin_http_requests_total",
			Help: "Total HTTP requests handled by admin API metric middleware",
		},
		[]string{"method", "route", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "goravel_admin_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"method", "route", "status"},
	)
)

func init() {
	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		httpRequestsTotal,
		httpRequestDuration,
	)
}

// Enabled reports whether Prometheus scrape endpoint is enabled.
func Enabled() bool {
	return facades.Config().GetBool("metrics.enabled", false)
}

// Token returns optional scrape bearer token (empty = open when enabled).
func Token() string {
	return strings.TrimSpace(facades.Config().GetString("metrics.token", ""))
}

// ObserveHTTP records one request sample for Prometheus.
func ObserveHTTP(method, route string, statusCode int, duration time.Duration) {
	if !Enabled() {
		return
	}
	method = strings.ToUpper(strings.TrimSpace(method))
	route = strings.TrimSpace(route)
	if method == "" || route == "" {
		return
	}
	status := strconv.Itoa(statusCode)
	httpRequestsTotal.WithLabelValues(method, route, status).Inc()
	httpRequestDuration.WithLabelValues(method, route, status).Observe(duration.Seconds())
}

// Handler returns the Prometheus scrape handler.
func Handler() http.Handler {
	return promhttp.HandlerFor(registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
}
