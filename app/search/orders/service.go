package orders

import (
	"context"
	"encoding/json"
	"fmt"

	"goravel/app/dto"
	orderrepo "goravel/app/repositories"
	"goravel/app/search"
)

// SyncEnabled 订单是否同步到当前搜索驱动。
func SyncEnabled() bool {
	return search.OrdersSyncEnabled()
}

// IndexName 订单索引短名。
func IndexName() string {
	return search.OrdersIndexShortName()
}

// Push 将单条订单 index 或 delete 到当前引擎。
func Push(ctx context.Context, orderID uint, orderNoHint string, op string) error {
	engine, err := search.Resolve()
	if err != nil {
		return err
	}
	index := IndexName()

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
func SearchMyOrders(ctx context.Context, userID uint, keyword string, page, pageSize int, createdAtGTE, createdAtLTE *string) (total int64, items []dto.OrderSearchListItem, err error) {
	engine, err := search.Resolve()
	if err != nil {
		return 0, nil, err
	}

	filters := []search.Filter{
		{Field: "user_id", Op: "term", Value: userID},
	}
	if createdAtGTE != nil {
		filters = append(filters, search.Filter{Field: "created_at", Op: "gte", Value: *createdAtGTE})
	}
	if createdAtLTE != nil {
		filters = append(filters, search.Filter{Field: "created_at", Op: "lte", Value: *createdAtLTE})
	}

	res, err := engine.Search(ctx, IndexName(), search.SearchRequest{
		Keyword:      keyword,
		SearchFields: []string{"order_no^2", "product_names", "remark"},
		Filters:      filters,
		Page:         page,
		PageSize:     pageSize,
		SortField:    "created_at",
		SortDesc:     true,
	})
	if err != nil {
		return 0, nil, err
	}

	items = make([]dto.OrderSearchListItem, 0, len(res.Hits))
	for _, hit := range res.Hits {
		var item dto.OrderSearchListItem
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

// InitIndex 显式初始化订单索引。
func InitIndex(ctx context.Context) error {
	engine, err := search.Resolve()
	if err != nil {
		return err
	}
	return engine.EnsureIndex(ctx, IndexName())
}
