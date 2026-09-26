package routes

import (
	"goravel/app/facades"
	"goravel/app/http/controllers"
)

func Web() {

	// facades.Route().Middleware(httpmiddleware.Throttle("testResponse")).Get("/", func(ctx http.Context) http.Response {
	// 	return ctx.Response().Json(http.StatusOK, http.Json{
	// 		"version": "1.0.0",
	// 	})
	// })

	// Swagger is disabled by default in production. Enable with SWAGGER_ENABLED=true
	// only for controlled environments.
	if facades.Config().GetBool("swagger.enabled", false) {
		swaggerController := controllers.NewSwaggerController()
		facades.Route().Get("/swagger/*any", swaggerController.Index)
	}

	healthController := controllers.NewHealthController()
	metricsController := controllers.NewMetricsController()
	// Liveness: process up (LB / k8s livenessProbe)
	facades.Route().Get("/health", healthController.Live)
	// Readiness: DB (+ Redis when required). Prefer this for k8s readinessProbe.
	facades.Route().Get("/ready", healthController.Ready)
	facades.Route().Get("/health/ready", healthController.Ready)
	// Prometheus scrape (METRICS_ENABLED=true)
	facades.Route().Get("/metrics", metricsController.Index)
}
