package config

import (
	"github.com/goravel/framework/facades"
)

func init() {
	config := facades.Config()
	config.Add("meilisearch", map[string]any{
		"host":   config.Env("MEILISEARCH_HOST", "http://127.0.0.1:7700"),
		"api_key": config.Env("MEILISEARCH_API_KEY", ""),
		"index_prefix": config.Env("MEILISEARCH_INDEX_PREFIX", ""),
	})
}
