package controllers

import (
	"github.com/goravel/framework/contracts/http"

	"goravel/app/health"
)

type HealthController struct{}

func NewHealthController() *HealthController {
	return &HealthController{}
}

// Live is a cheap liveness probe (process up). GET /health
func (c *HealthController) Live(ctx http.Context) http.Response {
	report := health.Live()
	payload := http.Json{
		"status":    report.Status,
		"timestamp": report.Timestamp,
	}
	if report.App != "" {
		payload["app"] = report.App
	}
	if report.Env != "" {
		payload["env"] = report.Env
	}
	if report.Version != "" {
		payload["version"] = report.Version
	}
	return ctx.Response().Json(http.StatusOK, payload)
}

// Ready checks DB (+ Redis when cache/queue require it). GET /ready
// Returns 503 when any required check fails (for k8s readiness / LB).
func (c *HealthController) Ready(ctx http.Context) http.Response {
	report := health.Ready(ctx)
	status := http.StatusOK
	if report.Status != "ready" {
		status = http.StatusServiceUnavailable
	}
	health.MaybeAlertNotReady(report)
	return ctx.Response().Json(status, http.Json{
		"status":    report.Status,
		"timestamp": report.Timestamp,
		"checks":    report.Checks,
	})
}
