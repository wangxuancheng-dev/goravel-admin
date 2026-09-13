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
			router.Get("tenants/settings", tenantController.Settings)
			router.Get("tenants/queue-status", tenantController.QueueStatus)
			router.Get("tenants/export", tenantController.Export)
			router.Get("tenant-op-logs", tenantController.OpLogsIndex)
			router.Post("tenants/migrate-batch", tenantController.MigrateBatch)
			router.Post("tenants/ops-batch", tenantController.OpsBatch)
			router.Get("tenants/{id}", tenantController.Show)
			router.Post("tenants", tenantController.Store)
			router.Put("tenants/{id}", tenantController.Update)
			router.Put("tenants/{id}/status", tenantController.UpdateStatus)
			router.Delete("tenants/{id}", tenantController.Destroy)
			router.Post("tenants/{id}/undelete", tenantController.Undelete)
			router.Delete("tenants/{id}/force", tenantController.ForceDestroy)
			router.Post("tenants/{id}/purge", tenantController.Purge)
			router.Post("tenants/{id}/ping", tenantController.Ping)
			router.Get("tenants/{id}/overview", tenantController.Overview)
			router.Get("tenants/{id}/op-logs", tenantController.OpLogs)
			router.Get("tenants/{id}/login-links", tenantController.LoginLinks)
			router.Post("tenants/{id}/migrate", tenantController.Migrate)
			router.Post("tenants/{id}/seed", tenantController.Seed)
			router.Post("tenants/{id}/backup", tenantController.Backup)
			router.Post("tenants/{id}/restore", tenantController.Restore)
			router.Get("tenants/{id}/backups", tenantController.ListBackups)
			router.Get("tenants/{id}/backups/download", tenantController.DownloadBackup)
			router.Post("tenants/{id}/backups/prune", tenantController.PruneBackups)
		})
	})
}
