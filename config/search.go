package config

import (
	"github.com/goravel/framework/facades"
)

func init() {
	config := facades.Config()
	config.Add("search", map[string]any{
		// elasticsearch | meilisearch | null
		"driver":  config.Env("SEARCH_DRIVER", "null"),
		"enabled": config.Env("SEARCH_ENABLED", false),
		// 同步逻辑队列名（与 bootstrap SearchQueueRunner 一致）
		"sync_queue":  config.Env("SEARCH_QUEUE", "search"),
		"sync_worker": config.Env("SEARCH_SYNC_WORKER", "auto"), // auto | true | false
		"outbox_enabled": config.Env("SEARCH_OUTBOX_ENABLED", true),
		"indexes": map[string]any{
			"orders": map[string]any{
				"sync_enabled": config.Env("SEARCH_SYNC_ORDERS", false),
				"name":         config.Env("SEARCH_ORDERS_INDEX", "orders"),
			},
			// 其他资源：在业务模块 RegisterDefinition + 增加 search.indexes.<key> 配置
		},
	})
}
