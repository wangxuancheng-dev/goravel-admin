package routes

import (
	"github.com/goravel/framework/contracts/route"
	httpmiddleware "github.com/goravel/framework/http/middleware"

	"goravel/app/facades"
	"goravel/app/http/controllers/platform"
	"goravel/app/http/middleware"
)

// Platform registers the landlord console API (platform DB only).
// Path prefix: /api/platform — never switches to a tenant connection.
func Platform() {
	authController := platform.NewAuthController()
	tenantController := platform.NewTenantController()
	healthController := platform.NewHealthController()
	passwordController := platform.NewPasswordController()

	facades.Route().Prefix("api/platform").Middleware(middleware.Domain(facades.Config().Get("domains.admin"))).Group(func(router route.Router) {
		router.Middleware(middleware.Lang(), middleware.RequireTenancy()).Group(func(router route.Router) {
			router.Middleware(httpmiddleware.Throttle("login")).Post("login", authController.Login)
		})

		router.Middleware(middleware.Lang(), middleware.RequireTenancy(), middleware.PlatformJwt()).Group(func(router route.Router) {
			router.Get("info", authController.Info)
			router.Post("logout", authController.Logout)
			router.Get("health", healthController.Index)
			router.Put("password", passwordController.Update)

			router.Get("tenants", tenantController.Index)
			router.Get("tenants/ops-summary", tenantController.OpsSummary)
			router.Post("tenants/migrate-batch", tenantController.MigrateBatch)
			router.Get("tenants/{id}", tenantController.Show)
			router.Post("tenants", tenantController.Store)
			router.Put("tenants/{id}", tenantController.Update)
			router.Put("tenants/{id}/status", tenantController.UpdateStatus)
			router.Post("tenants/{id}/ping", tenantController.Ping)
			router.Post("tenants/{id}/migrate", tenantController.Migrate)
			router.Post("tenants/{id}/seed", tenantController.Seed)
			router.Post("tenants/{id}/backup", tenantController.Backup)
		})
	})
}
