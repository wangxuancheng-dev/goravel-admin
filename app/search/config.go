package search

import (
	"strings"

	"github.com/goravel/framework/facades"
)

// Driver 返回当前搜索驱动名（elasticsearch / meilisearch / null）。
func Driver() string {
	d := strings.ToLower(strings.TrimSpace(facades.Config().GetString("search.driver", DriverNull)))
	if d == "" {
		return DriverNull
	}
	return d
}

// Enabled 搜索总开关。
func Enabled() bool {
	return facades.Config().GetBool("search.enabled", false) && Driver() != DriverNull
}

// SyncQueue 同步任务逻辑队列名。
func SyncQueue() string {
	q := strings.TrimSpace(facades.Config().GetString("search.sync_queue", "search"))
	if q == "" {
		return "search"
	}
	return q
}

// SyncWorkerMode auto | true | false
func SyncWorkerMode() string {
	return strings.ToLower(strings.TrimSpace(facades.Config().GetString("search.sync_worker", "auto")))
}

// OutboxEnabled 是否写同步 outbox。
func OutboxEnabled() bool {
	return facades.Config().GetBool("search.outbox_enabled", true)
}

// OrdersSyncEnabled 订单是否同步到当前搜索引擎。
func OrdersSyncEnabled() bool {
	if !Enabled() {
		return false
	}
	return facades.Config().GetBool("search.indexes.orders.sync_enabled", false)
}

// OrdersIndexShortName 订单索引短名（不含前缀）。
func OrdersIndexShortName() string {
	n := strings.TrimSpace(facades.Config().GetString("search.indexes.orders.name", "orders"))
	if n == "" {
		return "orders"
	}
	return n
}

// ShouldRunQueueWorker 是否启动搜索同步专用队列 Worker。
func ShouldRunQueueWorker() bool {
	if !Enabled() {
		return false
	}
	switch SyncWorkerMode() {
	case "false", "0", "no", "off":
		return false
	case "true", "1", "yes", "on":
		return true
	default:
		return OrdersSyncEnabled()
	}
}
