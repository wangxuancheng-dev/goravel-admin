package controllers

import (
	"bytes"
	"net/http"
	"strings"

	contractshttp "github.com/goravel/framework/contracts/http"

	"goravel/app/metrics"
)

type MetricsController struct{}

func NewMetricsController() *MetricsController {
	return &MetricsController{}
}

// Index exposes Prometheus scrape metrics when METRICS_ENABLED=true.
func (c *MetricsController) Index(ctx contractshttp.Context) contractshttp.Response {
	if !metrics.Enabled() {
		return ctx.Response().Data(http.StatusNotFound, "text/plain; charset=utf-8", []byte("metrics disabled\n"))
	}

	expected := metrics.Token()
	if expected != "" {
		got := strings.TrimSpace(ctx.Request().Header("Authorization", ""))
		if strings.HasPrefix(strings.ToLower(got), "bearer ") {
			got = strings.TrimSpace(got[7:])
		}
		if got == "" {
			got = strings.TrimSpace(ctx.Request().Query("token", ""))
		}
		if got != expected {
			return ctx.Response().Data(http.StatusUnauthorized, "text/plain; charset=utf-8", []byte("unauthorized\n"))
		}
	}

	var buf bytes.Buffer
	writer := &metricsResponseWriter{header: make(http.Header), body: &buf, code: http.StatusOK}
	req, _ := http.NewRequest(http.MethodGet, "/metrics", nil)
	metrics.Handler().ServeHTTP(writer, req)

	contentType := writer.header.Get("Content-Type")
	if contentType == "" {
		contentType = "text/plain; version=0.0.4; charset=utf-8"
	}
	return ctx.Response().Data(writer.code, contentType, buf.Bytes())
}

type metricsResponseWriter struct {
	header http.Header
	body   *bytes.Buffer
	code   int
}

func (w *metricsResponseWriter) Header() http.Header         { return w.header }
func (w *metricsResponseWriter) Write(p []byte) (int, error) { return w.body.Write(p) }
func (w *metricsResponseWriter) WriteHeader(statusCode int)  { w.code = statusCode }
