package middleware

import (
	"github.com/goravel/framework/contracts/http"

	"goravel/app/http/response"
	"goravel/app/utils"
)

func moduleEnabled(enabled func() bool, code string) http.Middleware {
	return newMiddleware("module_enabled_"+code, func(ctx http.Context) {
		if enabled() {
			ctx.Request().Next()
			return
		}
		response.Abort(ctx, http.StatusForbidden, code)
	})
}

// OrdersModule blocks requests when MODULE_ORDERS_ENABLED=false.
func OrdersModule() http.Middleware {
	return moduleEnabled(utils.OrdersEnabled, "module_orders_disabled")
}

// PaymentsModule blocks requests when MODULE_PAYMENTS_ENABLED=false.
func PaymentsModule() http.Middleware {
	return moduleEnabled(utils.PaymentsEnabled, "module_payments_disabled")
}

// ScheduleDemoModule blocks requests when MODULE_SCHEDULE_DEMO_ENABLED=false.
func ScheduleDemoModule() http.Middleware {
	return moduleEnabled(utils.ScheduleDemoEnabled, "module_schedule_demo_disabled")
}
