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
			filtered := []string{
				"about", "key:generate",
				"schedule:*", "queue:*", "migrate*",
				"db:*", "cache:*", "config:*", "lang:*",
				"app:*", "order:*", "payment:*", "search:*", "token:*",
			}
			if !facades.Config().GetBool("app.allow_maintenance_commands", false) {
				filtered = append([]string{"up", "down"}, filtered...)
			}
			return filtered
		}).
		WithConfig(config.Boot).
		Create()
}
