package providers

import (
	"fmt"

	"github.com/goravel/framework/contracts/foundation"

	"goravel/app/binding"
	"goravel/app/search"
	esdriver "goravel/app/search/drivers/elasticsearch"
	meili "goravel/app/search/drivers/meilisearch"
	searchorders "goravel/app/search/orders"
	"goravel/app/services"
)

// SearchServiceProvider registers the switchable search.Engine (SEARCH_DRIVER)
// and injects order loaders into search/orders.
// Domain: app/search (+ drivers, orders adapter). App services own DB reads.
// To extract later: move search/ to a module; keep this provider as the wiring point.
type SearchServiceProvider struct{}

func (r *SearchServiceProvider) Register(app foundation.Application) {
	search.SetResolver(func() (search.Engine, error) {
		return buildEngine(app)
	})

	app.Singleton(binding.SearchEngine, func(app foundation.Application) (any, error) {
		return buildEngine(app)
	})
}

func (r *SearchServiceProvider) Boot(app foundation.Application) {
	searchorders.SetOrderLoaders(services.FindOrderByID, services.FindOrderWithDetails)
}

func buildEngine(app foundation.Application) (search.Engine, error) {
	cfg := app.MakeConfig()
	if !search.Enabled() {
		return search.NewNullEngine(), nil
	}
	switch search.Driver() {
	case search.DriverElasticsearch:
		return esdriver.NewEngine(cfg)
	case search.DriverMeilisearch:
		return meili.NewEngine(cfg)
	case search.DriverNull:
		return search.NewNullEngine(), nil
	default:
		return nil, fmt.Errorf("unknown search driver: %s", search.Driver())
	}
}
