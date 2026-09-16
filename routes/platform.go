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
			// Edge on-demand TLS ask (no JWT); host must be active + ssl_mode=edge.
			router.Get("public/tls-allow", tenantController.TLSAllow)
		})

		router.Middleware(middleware.Lang(), middleware.RequireTenancy(), middleware.PlatformJwt()).Group(func(router route.Router) {
			// Readable by owner + viewer
			router.Get("info", authController.Info)
			router.Post("logout", authController.Logout)
			router.Get("health", healthController.Index)
			router.Get("ops/overview", tenantController.OpsOverview)
			router.Put("password", passwordController.Update)

			router.Get("tenants", tenantController.Index)
			router.Get("tenants/ops-summary", tenantController.OpsSummary)
			router.Get("tenants/settings", tenantController.Settings)
			router.Get("tenants/queue-status", tenantController.QueueStatus)
			router.Get("tenants/export", tenantController.Export)
			router.Get("tenant-op-logs", tenantController.OpLogsIndex)
			router.Get("tenants/{id}", tenantController.Show)
			router.Post("tenants/{id}/ping", tenantController.Ping)
			router.Get("tenants/{id}/overview", tenantController.Overview)
			router.Get("tenants/{id}/op-logs", tenantController.OpLogs)
			router.Get("tenants/{id}/login-links", tenantController.LoginLinks)
			router.Get("tenants/{id}/domains", tenantController.Domains)
			router.Get("tenants/{id}/backups", tenantController.ListBackups)
			router.Get("tenants/{id}/backups/download", tenantController.DownloadBackup)

			// Mutations require owner
			router.Middleware(middleware.PlatformOwner()).Group(func(router route.Router) {
				router.Post("tenants/migrate-batch", tenantController.MigrateBatch)
				router.Post("tenants/ops-batch", tenantController.OpsBatch)
				router.Post("tenants/onboard", tenantController.Onboard)
				router.Post("tenants/health-inspect", tenantController.HealthInspect)
				router.Post("tenants", tenantController.Store)
				router.Put("tenants/{id}", tenantController.Update)
				router.Put("tenants/{id}/status", tenantController.UpdateStatus)
				router.Put("tenants/{id}/maintenance", tenantController.UpdateMaintenance)
				router.Delete("tenants/{id}", tenantController.Destroy)
				router.Post("tenants/{id}/undelete", tenantController.Undelete)
				router.Delete("tenants/{id}/force", tenantController.ForceDestroy)
				router.Post("tenants/{id}/purge", tenantController.Purge)
				router.Post("tenants/{id}/migrate", tenantController.Migrate)
				router.Post("tenants/{id}/seed", tenantController.Seed)
				router.Post("tenants/{id}/backup", tenantController.Backup)
				router.Post("tenants/{id}/restore", tenantController.Restore)
				router.Post("tenants/{id}/backups/prune", tenantController.PruneBackups)
				router.Post("tenants/{id}/domains", tenantController.StoreDomain)
				router.Post("tenants/{id}/domains/{domainId}/verify", tenantController.VerifyDomain)
				router.Put("tenants/{id}/domains/{domainId}/primary", tenantController.SetPrimaryDomain)
				router.Put("tenants/{id}/domains/{domainId}/disable", tenantController.DisableDomain)
				router.Delete("tenants/{id}/domains/{domainId}", tenantController.DestroyDomain)
			})
		})
	})
}
