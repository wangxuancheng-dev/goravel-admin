package services

import (
	"context"
	"time"

	"github.com/goravel/framework/contracts/database/orm"
	"github.com/goravel/framework/contracts/http"

	"goravel/app/orders"
)

// OrderFilters re-exports the domain type for backward-compatible imports.
type OrderFilters = orders.Filters

// ApplyOrderFiltersToQuery 只负责通用筛选（不包含时间范围），供列表查询/导出复用，避免重复/不一致。
func ApplyOrderFiltersToQuery(query orm.Query, filters OrderFilters) orm.Query {
	return orders.ApplyFiltersToQuery(query, filters)
}

// BuildOrderQuery 构建订单分表查询（包含时间范围 + 通用筛选），供列表查询/导出复用。
func BuildOrderQuery(ctx context.Context, tableName string, filters OrderFilters) orm.Query {
	return orders.BuildQuery(ctx, tableName, filters)
}

// GetOrderDetailsTableFromOrdersTable 从订单分表名获取对应的订单详情分表名
// ordersTableName: 订单分表名，如 "orders_202501"
// 返回: 订单详情分表名，如 "order_details_202501"
func GetOrderDetailsTableFromOrdersTable(ordersTableName string) string {
	return orders.DetailsTableFromOrdersTable(ordersTableName)
}

// ParseOrderListTimeRange 解析后台订单列表时间；开始为空默认近 7 天，结束为空表示无上界。
// 错误返回可直接用作 response message key（invalid_start_time / invalid_end_time）。
func ParseOrderListTimeRange(startTimeStr, endTimeStr string) (time.Time, time.Time, error) {
	return orders.ParseListTimeRange(startTimeStr, endTimeStr)
}

// BuildOrderFiltersFromHTTP 从 query/body 构建订单列表/导出筛选（含默认时间）。
func BuildOrderFiltersFromHTTP(ctx http.Context) (OrderFilters, error) {
	return orders.BuildFiltersFromHTTP(ctx)
}
