package orders

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	orderrepo "goravel/app/repositories"
	"goravel/app/search"
)

// 可移植检索字段（纯字段名；加权由 FieldBoosts 表达，由各驱动自行翻译）。
var (
	orderSearchFields = []string{"order_no", "product_names", "remark"}
	orderFieldBoosts  = map[string]float64{"order_no": 2}
	adminSortFields   = map[string]bool{
		"created_at": true,
		"updated_at": true,
		"amount":     true,
		"id":         true,
		"status":     true,
		"order_no":   true,
		"user_id":    true,
	}
)

// SyncEnabled 订单是否同步到当前搜索驱动。
func SyncEnabled() bool {
	return search.OrdersSyncEnabled()
}

// QueryEnabled 订单列表/检索是否应走搜索引擎（同步开启且当前驱动检索可用）。
func QueryEnabled() bool {
	if !SyncEnabled() {
		return false
	}
	engine, err := search.Resolve()
	if err != nil || engine == nil {
		return false
	}
	return search.EngineQueryReady(engine)
}

// IndexName 订单索引短名（按 ctx 租户隔离）。
func IndexName(ctx context.Context) string {
	return search.OrdersIndexShortNameFor(ctx)
}

// Push 将单条订单 index 或 delete 到当前引擎。
func Push(ctx context.Context, orderID uint, orderNoHint string, op string) error {
	engine, err := search.Resolve()
	if err != nil {
		return err
	}
	index := IndexName(ctx)
	if strings.TrimSpace(index) == "" {
		return fmt.Errorf("search index requires tenant binding when tenancy is enabled")
	}

	if op == "delete" {
		orderNo := orderNoHint
		if orderNo == "" {
			order, err := orderrepo.FindOrderByID(ctx, orderID)
			if err != nil {
				return err
			}
			orderNo = order.OrderNo
		}
		if orderNo == "" {
			return fmt.Errorf("order_no required for search delete")
		}
		return engine.Delete(ctx, index, orderNo)
	}

	if err := engine.EnsureIndex(ctx, index); err != nil {
		return err
	}
	order, details, err := orderrepo.FindOrderWithDetails(ctx, orderID, orderNoHint)
	if err != nil {
		return err
	}
	if order.OrderNo == "" {
		return fmt.Errorf("order_no required for search indexing")
	}
	return engine.Index(ctx, index, order.OrderNo, Document(order, details))
}

// SearchMyOrders 当前用户订单检索（引擎路径）。
func SearchMyOrders(ctx context.Context, userID uint, keyword string, page, pageSize int, createdAtGTE, createdAtLTE *string) (total int64, items []ListItem, err error) {
	filters := []search.Filter{
		{Field: "user_id", Op: "term", Value: userID},
	}
	if createdAtGTE != nil {
		filters = append(filters, search.Filter{Field: "created_at", Op: "gte", Value: *createdAtGTE})
	}
	if createdAtLTE != nil {
		filters = append(filters, search.Filter{Field: "created_at", Op: "lte", Value: *createdAtLTE})
	}

	return searchOrders(ctx, search.SearchRequest{
		Keyword:      keyword,
		SearchFields: orderSearchFields,
		FieldBoosts:  orderFieldBoosts,
		Filters:      filters,
		Page:         page,
		PageSize:     pageSize,
		SortField:    "created_at",
		SortDesc:     true,
	})
}

// SearchAdminOrders 后台订单列表检索（引擎路径；复杂筛选/深翻页优先走索引）。
// 仅使用可移植 Filter/SearchRequest；调用方应在 QueryEnabled() 为 true 时使用，失败时回退分表 DB。
func SearchAdminOrders(ctx context.Context, userID uint, orderNo, status, keyword string, minAmount, maxAmount float64, page, pageSize int, createdAtGTE, createdAtLTE *string, sortField string, sortDesc bool) (total int64, items []ListItem, err error) {
	filters := make([]search.Filter, 0, 8)
	if userID > 0 {
		filters = append(filters, search.Filter{Field: "user_id", Op: "term", Value: userID})
	}
	if orderNo != "" {
		filters = append(filters, search.Filter{Field: "order_no", Op: "term", Value: orderNo})
	}
	if status != "" {
		filters = append(filters, search.Filter{Field: "status", Op: "term", Value: status})
	}
	if minAmount > 0 {
		filters = append(filters, search.Filter{Field: "amount", Op: "gte", Value: minAmount})
	}
	if maxAmount > 0 {
		filters = append(filters, search.Filter{Field: "amount", Op: "lte", Value: maxAmount})
	}
	if createdAtGTE != nil {
		filters = append(filters, search.Filter{Field: "created_at", Op: "gte", Value: *createdAtGTE})
	}
	if createdAtLTE != nil {
		filters = append(filters, search.Filter{Field: "created_at", Op: "lte", Value: *createdAtLTE})
	}

	sortField = strings.TrimSpace(sortField)
	if sortField == "" || !adminSortFields[sortField] {
		sortField = "created_at"
		sortDesc = true
	}

	return searchOrders(ctx, search.SearchRequest{
		Keyword:      keyword,
		SearchFields: orderSearchFields,
		FieldBoosts:  orderFieldBoosts,
		Filters:      filters,
		Page:         page,
		PageSize:     pageSize,
		SortField:    sortField,
		SortDesc:     sortDesc,
	})
}

func searchOrders(ctx context.Context, req search.SearchRequest) (total int64, items []ListItem, err error) {
	engine, err := search.Resolve()
	if err != nil {
		return 0, nil, err
	}
	if !search.EngineQueryReady(engine) {
		return 0, nil, fmt.Errorf("%w: orders query not ready on driver %s", search.ErrUnsupported, engine.Name())
	}

	index := IndexName(ctx)
	if strings.TrimSpace(index) == "" {
		return 0, nil, fmt.Errorf("search index requires tenant binding when tenancy is enabled")
	}

	res, err := engine.Search(ctx, index, req)
	if err != nil {
		return 0, nil, err
	}

	items = make([]ListItem, 0, len(res.Hits))
	for _, hit := range res.Hits {
		var item ListItem
		b, err := json.Marshal(hit)
		if err != nil {
			continue
		}
		if err := json.Unmarshal(b, &item); err != nil {
			continue
		}
		items = append(items, item)
	}
	return res.Total, items, nil
}

// IsUnsupported 判断是否为当前驱动尚未实现的能力（应回退 DB）。
func IsUnsupported(err error) bool {
	return err != nil && errors.Is(err, search.ErrUnsupported)
}

// InitIndex 显式初始化订单索引。
func InitIndex(ctx context.Context) error {
	engine, err := search.Resolve()
	if err != nil {
		return err
	}
	return engine.EnsureIndex(ctx, IndexName(ctx))
}
