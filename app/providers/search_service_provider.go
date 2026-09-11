package providers

import (
	"fmt"

	"github.com/goravel/framework/contracts/foundation"

	"goravel/app/binding"
	"goravel/app/clients"
	"goravel/app/search"
	esdriver "goravel/app/search/drivers/elasticsearch"
	meili "goravel/app/search/drivers/meilisearch"
)

// SearchServiceProvider 注册可切换的搜索 Engine（及 ES 客户端，当 driver=elasticsearch）。
type SearchServiceProvider struct{}

func (r *SearchServiceProvider) Register(app foundation.Application) {
	search.SetResolver(func() (search.Engine, error) {
		return buildEngine(app)
	})

	app.Singleton(binding.SearchEngine, func(app foundation.Application) (any, error) {
		return buildEngine(app)
	})

	if search.Driver() == search.DriverElasticsearch && search.Enabled() {
		app.Singleton(binding.ElasticsearchClient, func(app foundation.Application) (any, error) {
			return clients.NewElasticsearchClient(app.MakeConfig(), "")
		})
	}
}

func (r *SearchServiceProvider) Boot(app foundation.Application) {}

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
