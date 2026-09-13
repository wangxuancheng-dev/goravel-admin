package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/goravel/framework/contracts/database/orm"
	"github.com/goravel/framework/contracts/http"
	"github.com/spf13/cast"

	appfacades "goravel/app/facades"
	"goravel/app/http/helpers"
	"goravel/app/utils"
)

// OrderFilters are shared order list / export query criteria.
type OrderFilters struct {
	UserID    uint      // 用户ID（0表示不筛选）
	OrderNo   string    // 订单号（精确匹配，后台列表）
	Keyword   string    // 关键词（订单号、备注 LIKE，供 C 端搜索等）
	Status    string    // 订单状态
	MinAmount float64   // 最小金额（0表示不筛选）
	MaxAmount float64   // 最大金额（0表示不筛选）
	StartTime time.Time // 开始时间
	EndTime   time.Time // 结束时间
	OrderBy   string    // 排序字段（格式：字段:asc/desc，如：created_at:desc）
}

// ApplyOrderFiltersToQuery applies non-time filters for list / export reuse.
func ApplyOrderFiltersToQuery(query orm.Query, filters OrderFilters) orm.Query {
	if filters.UserID > 0 {
		query = query.Where("user_id = ?", filters.UserID)
	}

	if filters.OrderNo != "" {
		query = query.Where("order_no = ?", filters.OrderNo)
	}

	if kw := strings.TrimSpace(filters.Keyword); kw != "" {
		pattern := "%" + kw + "%"
		query = query.Where("(order_no LIKE ? OR remark LIKE ?)", pattern, pattern)
	}

	if filters.Status != "" {
		query = query.Where("status = ?", filters.Status)
	}

	if filters.MinAmount > 0 {
		query = query.Where("amount >= ?", filters.MinAmount)
	}
	if filters.MaxAmount > 0 {
		query = query.Where("amount <= ?", filters.MaxAmount)
	}

	return query
}

// BuildOrderQuery builds a sharded order query (time range + common filters).
func BuildOrderQuery(ctx context.Context, tableName string, filters OrderFilters) orm.Query {
	query := appfacades.OrmQuery(ctx).Table(tableName)

	if !filters.StartTime.IsZero() {
		query = query.Where("created_at >= ?", filters.StartTime)
	}
	if !filters.EndTime.IsZero() {
		query = query.Where("created_at <= ?", filters.EndTime)
	}

	return ApplyOrderFiltersToQuery(query, filters)
}

// OrderDetailsTableFromOrdersTable maps orders_YYYYMM → order_details_YYYYMM.
func OrderDetailsTableFromOrdersTable(ordersTableName string) string {
	return strings.Replace(ordersTableName, "orders_", "order_details_", 1)
}

// ParseOrderListTimeRange parses admin order list times; empty start defaults to last 7 days,
// empty end means no upper bound. Errors are response message keys.
func ParseOrderListTimeRange(startTimeStr, endTimeStr string) (time.Time, time.Time, error) {
	var startTime, endTime time.Time
	var err error

	if startTimeStr == "" {
		startTime = time.Now().UTC().AddDate(0, 0, -7)
	} else {
		startTime, err = utils.ParseDateTime(startTimeStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid_start_time")
		}
	}

	if endTimeStr == "" {
		endTime = time.Time{}
	} else {
		endTime, err = utils.ParseDateTime(endTimeStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid_end_time")
		}
	}

	return startTime, endTime, nil
}

// BuildOrderFiltersFromHTTP builds list/export filters from query/body (including default time).
func BuildOrderFiltersFromHTTP(ctx http.Context) (OrderFilters, error) {
	startTime, endTime, err := ParseOrderListTimeRange(
		helpers.GetTimeInputOrQueryParam(ctx, "start_time"),
		helpers.GetTimeInputOrQueryParam(ctx, "end_time"),
	)
	if err != nil {
		return OrderFilters{}, err
	}

	return OrderFilters{
		UserID:    cast.ToUint(ctx.Request().Input("user_id", ctx.Request().Query("user_id", "0"))),
		OrderNo:   ctx.Request().Input("order_no", ctx.Request().Query("order_no", "")),
		Status:    ctx.Request().Input("status", ctx.Request().Query("status", "")),
		MinAmount: cast.ToFloat64(ctx.Request().Input("min_amount", ctx.Request().Query("min_amount", "0"))),
		MaxAmount: cast.ToFloat64(ctx.Request().Input("max_amount", ctx.Request().Query("max_amount", "0"))),
		StartTime: startTime,
		EndTime:   endTime,
		OrderBy:   ctx.Request().Input("order_by", ctx.Request().Query("order_by", "")),
	}, nil
}
