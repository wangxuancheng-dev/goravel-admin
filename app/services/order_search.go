package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/goravel/framework/facades"

	apperrors "goravel/app/errors"
	orderrepo "goravel/app/repositories"
	"goravel/app/search"
	searchorders "goravel/app/search/orders"
	"goravel/app/utils"
	"goravel/app/utils/errorlog"
)

func (s *OrderServiceImpl) getOrdersWithDetailsFromSearch(filters OrderFilters, page, pageSize int) ([]OrderWithDetails, int64, error) {
	valid, err := utils.ValidateTimeRange(filters.StartTime, filters.EndTime)
	if !valid {
		return nil, 0, err
	}
	if err := utils.ValidateShardingDeepPagination(page, pageSize); err != nil {
		maxRows := utils.GetMaxUnionLimitPerTable()
		if m, ok := utils.DeepPaginationMaxFromError(err); ok {
			maxRows = m
		}
		return nil, 0, apperrors.ErrDeepPaginationExceeded.WithParams(map[string]any{"max": maxRows})
	}

	var gte, lte *string
	if !filters.StartTime.IsZero() {
		v := utils.FormatDateTime(filters.StartTime)
		gte = &v
	}
	if !filters.EndTime.IsZero() {
		v := utils.FormatDateTime(filters.EndTime)
		lte = &v
	}

	sortField := "created_at"
	sortDesc := true
	if filters.OrderBy != "" {
		parts := strings.Split(filters.OrderBy, ":")
		if len(parts) == 2 {
			sortField = parts[0]
			sortDesc = strings.ToLower(parts[1]) != "asc"
		}
	}

	keyword := strings.TrimSpace(filters.Keyword)
	total, items, err := searchorders.SearchAdminOrders(
		s.ctx,
		filters.UserID,
		filters.OrderNo,
		filters.Status,
		keyword,
		filters.MinAmount,
		filters.MaxAmount,
		page,
		pageSize,
		gte,
		lte,
		sortField,
		sortDesc,
	)
	if err != nil {
		return nil, 0, err
	}
	if len(items) == 0 {
		return []OrderWithDetails{}, total, nil
	}

	result := make([]OrderWithDetails, 0, len(items))
	missed := 0
	for _, item := range items {
		order, details, err := orderrepo.FindOrderWithDetails(s.ctx, item.ID, item.OrderNo)
		if err != nil || order == nil {
			missed++
			continue
		}
		result = append(result, OrderWithDetails{
			Order:   *order,
			Details: details,
		})
	}
	// ????????????? DB ??????????????+?? total
	if len(result) == 0 && len(items) > 0 {
		return nil, 0, fmt.Errorf("order search hydrate failed: %d hits, all missing in db", len(items))
	}
	if missed > 0 {
		errorlog.Record(s.ctx, "order", "??????????", map[string]any{
			"missed": missed,
			"hits":   len(items),
		}, "??????????: missed=%d hits=%d", missed, len(items))
	}
	return result, total, nil
}

func orderWithDetailsToSearchListItem(o OrderWithDetails) searchorders.ListItem {
	names := make([]string, 0, len(o.Details))
	for _, d := range o.Details {
		names = append(names, d.ProductName)
	}
	return searchorders.ListItem{
		ID:           o.ID,
		OrderNo:      o.OrderNo,
		Amount:       o.Amount,
		Status:       o.Status,
		Remark:       o.Remark,
		CreatedAt:    o.CreatedAt.ToDateTimeString(),
		ProductNames: names,
	}
}

// searchMyOrdersFromDB C ?????????????? + ??? LIKE??
func (s *OrderServiceImpl) searchMyOrdersFromDB(userID uint, keyword string, page, pageSize int, tr searchorders.CreatedRange) ([]searchorders.ListItem, int64, error) {
	valid, err := utils.ValidateTimeRange(tr.DBStart, tr.DBEnd)
	if !valid {
		return nil, 0, err
	}

	filters := OrderFilters{
		UserID:    userID,
		StartTime: tr.DBStart,
		EndTime:   tr.DBEnd,
		Keyword:   strings.TrimSpace(keyword),
		OrderBy:   "created_at:desc",
	}
	// ???? DB????????????? GetOrdersWithDetails ????????
	rows, total, err := s.getOrdersWithDetailsFromDB(filters, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	out := make([]searchorders.ListItem, 0, len(rows))
	for i := range rows {
		out = append(out, orderWithDetailsToSearchListItem(rows[i]))
	}
	return out, total, nil
}

// SearchMyOrdersForUser C ???????????????????????????????????????????????????? 3 ????????????
func (s *OrderServiceImpl) SearchMyOrdersForUser(ctx context.Context, userID uint, keyword string, page, pageSize int, tr searchorders.CreatedRange) ([]searchorders.ListItem, int64, error) {
	if searchorders.QueryEnabled() {
		total, items, err := searchorders.SearchMyOrders(ctx, userID, keyword, page, pageSize, tr.IndexGTE, tr.IndexLTE)
		if err != nil {
			facades.Log().Warningf("order search engine failed (driver=%s), fallback to DB: %v", search.Driver(), err)
			return s.searchMyOrdersFromDB(userID, keyword, page, pageSize, tr)
		}
		return items, total, nil
	}

	return s.searchMyOrdersFromDB(userID, keyword, page, pageSize, tr)
}
