package config

import (
	"github.com/goravel/framework/facades"
)

func init() {
	config := facades.Config()
	// Elasticsearch 仅作为 search.driver=elasticsearch 时的连接与分词配置
	config.Add("elasticsearch", map[string]any{
		"default": config.Env("ELASTICSEARCH_CONNECTION", "default"),
		"connections": map[string]any{
			"default": map[string]any{
				"urls":                 config.Env("ELASTICSEARCH_URLS", "http://127.0.0.1:9200"),
				"username":             config.Env("ELASTICSEARCH_USERNAME", ""),
				"password":             config.Env("ELASTICSEARCH_PASSWORD", ""),
				"api_key":              config.Env("ELASTICSEARCH_API_KEY", ""),
				"cloud_id":             config.Env("ELASTICSEARCH_CLOUD_ID", ""),
				"insecure_skip_verify": config.Env("ELASTICSEARCH_INSECURE_SKIP_VERIFY", false),
			},
		},
		"index_prefix":           config.Env("ELASTICSEARCH_INDEX_PREFIX", ""),
		"demo_index":             config.Env("ELASTICSEARCH_DEMO_INDEX", "goravel_demo"),
		"orders_analyzer":        config.Env("ELASTICSEARCH_ORDERS_ANALYZER", "auto"),
		"orders_search_analyzer": config.Env("ELASTICSEARCH_ORDERS_SEARCH_ANALYZER", ""),
	})
}
