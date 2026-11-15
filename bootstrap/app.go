package bootstrap

import (
	"goravel/config"
	"goravel/routes"

	contractsfoundation "github.com/goravel/framework/contracts/foundation"
	contractsconfiguration "github.com/goravel/framework/contracts/foundation/configuration"
	"github.com/goravel/framework/facades"
	"github.com/goravel/framework/foundation"
	telemetryhttp "github.com/goravel/framework/telemetry/instrumentation/http"
)

func Boot() contractsfoundation.Application {
	return foundation.Setup().
		WithMiddleware(func(handler contractsconfiguration.Middleware) {
			handler.Append(telemetryhttp.Telemetry())
		}).
		WithMigrations(Migrations).
		WithSeeders(Seeders).
		WithRouting(func() {
			routes.Web()
			routes.Api()
			routes.Admin()
			routes.Platform()
			routes.Pprof()
		}).
		WithProviders(Providers).
		WithRunners(func() []contractsfoundation.Runner {
			return QueueRunners()
		}).
		WithCommandsFilter(func() []string {
			if facades.Config().GetString("app.env", "production") != "production" {
				return nil
			}
			return ProductionCommandsAllowlist(facades.Config().GetBool("app.allow_maintenance_commands", false))
		}).
		WithConfig(config.Boot).
		Create()
}
