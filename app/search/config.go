package search

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/goravel/framework/facades"

	"goravel/app/tenancy"
	"goravel/app/tenancyctx"
)

var tenantIndexSegment = regexp.MustCompile(`[^a-z0-9_]+`)

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

// OrdersIndexShortName 订单索引配置短名（不含租户段、不含 ES 前缀）。
func OrdersIndexShortName() string {
	n := strings.TrimSpace(facades.Config().GetString("search.indexes.orders.name", "orders"))
	if n == "" {
		return "orders"
	}
	return n
}

// OrdersIndexShortNameFor 返回带租户隔离段的索引短名（tenancy 开启且 ctx 已绑定时）。
// 例：acme_orders / t3_orders。
// tenancy 开启但未绑定租户时返回空字符串（fail-closed，避免写入共享 orders 索引）。
func OrdersIndexShortNameFor(ctx context.Context) string {
	base := OrdersIndexShortName()
	if !tenancy.Enabled() {
		return base
	}
	if code, ok := tenancyctx.CodeFrom(ctx); ok {
		seg := sanitizeIndexSegment(code)
		if seg != "" {
			return seg + "_" + base
		}
	}
	if id, ok := tenancyctx.IDFrom(ctx); ok {
		return fmt.Sprintf("t%d_%s", id, base)
	}
	return ""
}

// IsOrdersIndexShortName 判断短名是否为订单索引（含租户前缀形态）。
func IsOrdersIndexShortName(index string) bool {
	index = strings.TrimSpace(index)
	if index == "" {
		return false
	}
	base := OrdersIndexShortName()
	if index == base || index == "orders" {
		return true
	}
	return strings.HasSuffix(index, "_"+base) || strings.HasSuffix(index, "_orders")
}

func sanitizeIndexSegment(code string) string {
	s := strings.ToLower(strings.TrimSpace(code))
	s = tenantIndexSegment.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	if len(s) > 48 {
		s = s[:48]
	}
	return s
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
