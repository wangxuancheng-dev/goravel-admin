package services

import (
	"context"
	"errors"
	"fmt"
	appfacades "goravel/app/facades"
	"sort"
	"strings"
	"time"

	"github.com/goravel/framework/contracts/database/orm"
	"github.com/goravel/framework/facades"
	"github.com/oklog/ulid/v2"

	apperrors "goravel/app/errors"
	"goravel/app/models"
	orderrepo "goravel/app/repositories"
	"goravel/app/search"
	searchorders "goravel/app/search/orders"
	"goravel/app/support"
	"goravel/app/tenancy"
	"goravel/app/tenancyctx"
	"goravel/app/utils"
	"goravel/app/utils/errorlog"
)

const OrderCountThreshold int64 = 100000

func orderTenantID(ctx context.Context) uint {
	id, _ := tenancyctx.IDFrom(ctx)
	return id
}

type OrderService interface {
	// CreateOrder ????????????
	CreateOrder(userID uint, amount float64, products []OrderProduct, requestID string, remark string) (*models.Order, []models.OrderDetail, error)
	// GetOrderByID ??ID????
	GetOrderByID(orderID uint, orderTime time.Time) (*models.Order, []models.OrderDetail, error)
	// GetOrderByOrderNo ?????????????????????
	GetOrderByOrderNo(orderNo string) (*models.Order, []models.OrderDetail, error)
	// GetOrders ????????????3???
	GetOrders(filters OrderFilters, page, pageSize int) ([]models.Order, int64, error)
	// GetOrdersWithDetails ?????????????????3???
	GetOrdersWithDetails(filters OrderFilters, page, pageSize int) ([]OrderWithDetails, int64, error)
	// OrderToJSON ????????
	OrderToJSON(order models.Order) map[string]any
	// OrderDetailToJSON ????????
	OrderDetailToJSON(detail models.OrderDetail) map[string]any
	// OrderWithDetailsToJSON ??????????
	OrderWithDetailsToJSON(item *OrderWithDetails) map[string]any
	// GetAllOrdersForExport ????????????????3???????
	GetAllOrdersForExport(filters OrderFilters) ([]models.Order, error)
	// GetAllOrdersWithDetailsForExport ???????????????????3???????
	GetAllOrdersWithDetailsForExport(filters OrderFilters) ([]OrderWithDetails, error)
	// UpdateOrder ????????? order_no ??????
	UpdateOrder(orderID uint, orderTime time.Time, status string, remark string, orderNo ...string) error
	// UpdateOrderByOrderNo ????????????????
	UpdateOrderByOrderNo(orderNo string, status string, remark string) error
	// DeleteOrder ????????? order_no ??????
	DeleteOrder(orderID uint, orderTime time.Time, orderNo ...string) error
	// DeleteOrderByOrderNo ?????????
	DeleteOrderByOrderNo(orderNo string) error
	// GetOrdersCountInYear ????????????????????
	GetOrdersCountInYear() (int64, error)
	// SearchMyOrdersForUser C ????????????????????????????????????????????
	SearchMyOrdersForUser(ctx context.Context, userID uint, keyword string, page, pageSize int, tr searchorders.CreatedRange) ([]searchorders.ListItem, int64, error)
}

type OrderServiceImpl struct {
	ctx                  context.Context
	shardingService      ShardingService
	shardingQueryService ShardingQueryService
	countThreshold       int64 // count ??????
}

type OrderProduct struct {
	ProductID   uint    `json:"product_id"`
	ProductName string  `json:"product_name"`
	Price       float64 `json:"price"`
	Quantity    int     `json:"quantity"`
}

func NewOrderService(ctx context.Context) *OrderServiceImpl {
	service := &OrderServiceImpl{
		ctx:             ctx,
		shardingService: NewShardingService(ctx),
	}

	// ?????????
	service.shardingQueryService = NewShardingQueryService(ctx, ShardingQueryConfig{
		BaseTableName: "orders",
		GetColumns: func() string {
			return service.getOrderTableColumns()
		},
		UnionCollation: "utf8mb4_unicode_ci", // ?? MySQL UNION ? "Illegal mix of collations" ??
		StringColumns:  []string{"order_no", "status", "remark"},
		BuildWhereClause: func(filters any) (string, []any) {
			return service.buildOrderWhereClause(filters)
		},
		GetAllowedOrderFields: func() map[string]bool {
			return map[string]bool{
				"id":         true,
				"order_no":   true,
				"user_id":    true,
				"amount":     true,
				"status":     true,
				"created_at": true,
				"updated_at": true,
			}
		},
		DefaultOrderBy: "created_at:desc",
		ModuleName:     "order",
		CountThreshold: OrderCountThreshold, // ??????????????????????0????count
	})

	return service
}

// CreateOrder ??????? + ??????DDL ????????
func (s *OrderServiceImpl) CreateOrder(userID uint, amount float64, products []OrderProduct, requestID string, remark string) (*models.Order, []models.OrderDetail, error) {
	if requestID == "" {
		requestID = ulid.Make().String()
	}

	lockKey := tenancy.CacheKey(s.ctx, fmt.Sprintf("order:lock:%s", requestID))
	lockValue := fmt.Sprintf("%d_%d", userID, time.Now().Unix())

	var cachedValue string
	cacheResult := facades.Cache().Get(lockKey, &cachedValue)
	if cacheResult != nil && cachedValue != "" {
		return nil, nil, errors.New("??????????????")
	}

	if err := facades.Cache().Put(lockKey, lockValue, 5*time.Second); err != nil {
		errorlog.Record(s.ctx, "order", "?????", map[string]any{
			"user_id":    userID,
			"request_id": requestID,
			"lock_key":   lockKey,
			"error":      err.Error(),
		}, "?????: %v", err)
		return nil, nil, apperrors.ErrGetLockFailed.WithError(err)
	}
	defer func() {
		_ = facades.Cache().Forget(lockKey)
	}()

	now := time.Now().UTC()
	tableName := utils.GetShardingTableName("orders", now)
	detailTableName := utils.GetShardingTableName("order_details", now)

	// MySQL DDL ??????????????
	if err := s.shardingService.EnsureShardingTable(tableName, "orders"); err != nil {
		return nil, nil, err
	}
	if err := s.shardingService.EnsureShardingTable(detailTableName, "order_details"); err != nil {
		return nil, nil, err
	}

	var order *models.Order
	var details []models.OrderDetail
	err := appfacades.OrmTransaction(s.ctx, func(tx orm.Query) error {
		maxRetries := 3
		for i := range maxRetries {
			candidate := &models.Order{
				OrderNo: s.generateOrderNo(),
				UserID:  userID,
				Amount:  amount,
				Status:  models.OrderStatusPending,
				Remark:  remark,
			}
			if err := tx.Table(tableName).Create(candidate); err != nil {
				errStr := err.Error()
				if strings.Contains(errStr, "Duplicate entry") || strings.Contains(errStr, "1062") {
					if i == maxRetries-1 {
						errorlog.Record(s.ctx, "order", "?????????", map[string]any{
							"user_id":    userID,
							"request_id": requestID,
							"retries":    maxRetries,
						}, "?????????????")
						return apperrors.ErrGenerateOrderNoFailed
					}
					continue
				}
				errorlog.Record(s.ctx, "order", "??????", map[string]any{
					"user_id":    userID,
					"request_id": requestID,
					"amount":     amount,
					"error":      err.Error(),
				}, "??????: %v", err)
				return apperrors.ErrCreateOrderFailed.WithError(err)
			}
			order = candidate
			break
		}
		if order == nil || order.ID == 0 {
			return apperrors.ErrCreateOrderFailed
		}

		details = make([]models.OrderDetail, 0, len(products))
		for _, product := range products {
			detail := models.OrderDetail{
				OrderID:     order.ID,
				ProductID:   product.ProductID,
				ProductName: product.ProductName,
				Price:       product.Price,
				Quantity:    product.Quantity,
				Subtotal:    product.Price * float64(product.Quantity),
			}
			if err := tx.Table(detailTableName).Create(&detail); err != nil {
				errorlog.Record(s.ctx, "order", "????????", map[string]any{
					"order_id":   order.ID,
					"user_id":    userID,
					"product_id": product.ProductID,
					"error":      err.Error(),
				}, "????????: %v", err)
				return apperrors.ErrCreateOrderDetailFailed.WithError(err)
			}
			details = append(details, detail)
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	support.RequestOrderSearchSync(order.ID, order.OrderNo, "index", orderTenantID(s.ctx))
	return order, details, nil
}

// findOrderByID ?? orderrepo?? ES ???????????????????
func (s *OrderServiceImpl) findOrderByID(orderID uint, orderNo ...string) (*models.Order, error) {
	if len(orderNo) > 0 && orderNo[0] != "" {
		return orderrepo.FindOrderByID(s.ctx, orderID, orderNo[0])
	}
	return orderrepo.FindOrderByID(s.ctx, orderID)
}

// GetOrderByID ??ID????
func (s *OrderServiceImpl) GetOrderByID(orderID uint, orderTime time.Time) (*models.Order, []models.OrderDetail, error) {
	return orderrepo.FindOrderWithDetails(s.ctx, orderID, "")
}

// GetOrderByOrderNo ?????????????????????
func (s *OrderServiceImpl) GetOrderByOrderNo(orderNo string) (*models.Order, []models.OrderDetail, error) {
	return orderrepo.FindOrderWithDetails(s.ctx, 0, orderNo)
}

// GetOrders ?????????????????? max_union_limit_per_table ???
func (s *OrderServiceImpl) GetOrders(filters OrderFilters, page, pageSize int) ([]models.Order, int64, error) {
	// ?????????????
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

	// ???????????
	tableNames := utils.GetShardingTableNames("orders", filters.StartTime, filters.EndTime)
	if len(tableNames) == 0 {
		return []models.Order{}, 0, nil
	}

	// ?????????????
	if len(tableNames) == 1 {
		return s.querySingleTable(tableNames[0], filters, page, pageSize)
	}

	// ??????? UNION ALL ????????
	return s.queryMultipleTablesWithUnion(tableNames, filters, page, pageSize)
}

// buildShardingQuery ?????????????????????
func (s *OrderServiceImpl) buildShardingQuery(tableName string, filters OrderFilters) orm.Query {
	return BuildOrderQuery(s.ctx, tableName, filters)
}

// buildOrderWhereClause ??????? WHERE ??????????????
func (s *OrderServiceImpl) buildOrderWhereClause(filters any) (string, []any) {
	orderFilters, ok := filters.(OrderFilters)
	if !ok {
		return "", nil
	}

	var conditions []string
	var args []any

	// ????????
	if !orderFilters.StartTime.IsZero() {
		conditions = append(conditions, "created_at >= ?")
		args = append(args, orderFilters.StartTime)
	}
	if !orderFilters.EndTime.IsZero() {
		conditions = append(conditions, "created_at <= ?")
		args = append(args, orderFilters.EndTime)
	}

	// ??ID??
	if orderFilters.UserID > 0 {
		conditions = append(conditions, "user_id = ?")
		args = append(args, orderFilters.UserID)
	}

	// ???????
	if orderFilters.OrderNo != "" {
		// conditions = append(conditions, "order_no LIKE ?")
		// args = append(args, "%"+orderFilters.OrderNo+"%")
		conditions = append(conditions, "order_no = ?")
		args = append(args, orderFilters.OrderNo)
	}

	if kw := strings.TrimSpace(orderFilters.Keyword); kw != "" {
		pattern := "%" + kw + "%"
		conditions = append(conditions, "(order_no LIKE ? OR remark LIKE ?)")
		args = append(args, pattern, pattern)
	}

	// ??????
	if orderFilters.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, orderFilters.Status)
	}

	// ??????
	if orderFilters.MinAmount > 0 {
		conditions = append(conditions, "amount >= ?")
		args = append(args, orderFilters.MinAmount)
	}
	if orderFilters.MaxAmount > 0 {
		conditions = append(conditions, "amount <= ?")
		args = append(args, orderFilters.MaxAmount)
	}

	return strings.Join(conditions, " AND "), args
}

// getOrderTableColumns ????????????? UNION ALL ???
// ?????????????????????
// ??? Order ??????????????????????? payment_method?
func (s *OrderServiceImpl) getOrderTableColumns() string {
	// ?????? Order ????????
	// ??? Order ?????????????????????
	// ?????? payment_method????
	// 1. Order ????????
	// 2. ????????????
	// 3. ??????????????? Order ????????????????
	columns := []string{
		"id",
		"order_no",
		"user_id",
		"amount",
		"status",
		"remark",
		"created_at",
		"updated_at",
		"deleted_at",
	}
	return strings.Join(columns, ", ")
}

// queryMultipleTablesWithUnion ?? UNION ALL ??????
// ?????????????????????????
// ??????????
func (s *OrderServiceImpl) queryMultipleTablesWithUnion(tableNames []string, filters OrderFilters, page, pageSize int) ([]models.Order, int64, error) {
	var orders []models.Order
	total, err := s.shardingQueryService.QueryMultipleTables(tableNames, filters, page, pageSize, &orders)
	if err != nil {
		return nil, 0, err
	}
	return orders, total, nil
}

// queryMultipleTablesWithUnionForExport ?? UNION ALL ????????????????
// ??????????????????????
// ??????????
func (s *OrderServiceImpl) queryMultipleTablesWithUnionForExport(tableNames []string, filters OrderFilters) ([]models.Order, error) {
	var orders []models.Order
	err := s.shardingQueryService.QueryMultipleTablesForExport(tableNames, filters, &orders)
	if err != nil {
		return nil, err
	}
	return orders, nil
}

// sortOrders ?????????
func (s *OrderServiceImpl) sortOrders(orders []models.Order, orderBy string) {
	parts := strings.Split(orderBy, ":")
	field := "created_at"
	desc := true
	if len(parts) == 2 {
		field = parts[0]
		desc = strings.ToLower(parts[1]) == "desc"
	}

	// ?? sort.Slice ????
	sort.Slice(orders, func(i, j int) bool {
		var less bool
		switch field {
		case "id":
			less = orders[i].ID < orders[j].ID
		case "amount":
			less = orders[i].Amount < orders[j].Amount
		case "created_at":
			less = orders[i].CreatedAt.ToDateTimeString() < orders[j].CreatedAt.ToDateTimeString()
		case "updated_at":
			less = orders[i].UpdatedAt.ToDateTimeString() < orders[j].UpdatedAt.ToDateTimeString()
		default:
			less = orders[i].CreatedAt.ToDateTimeString() < orders[j].CreatedAt.ToDateTimeString()
		}
		if desc {
			return !less
		}
		return less
	})
}

// querySingleTable ??????
func (s *OrderServiceImpl) querySingleTable(tableName string, filters OrderFilters, page, pageSize int) ([]models.Order, int64, error) {
	// ???????????????????????? SQL 1146 ???
	if !utils.ShardingTableExistsCtx(s.ctx, tableName) {
		return []models.Order{}, 0, nil
	}

	// ????
	orderBy := filters.OrderBy
	if orderBy == "" {
		orderBy = "created_at:desc"
	}

	// ????????
	query := s.buildShardingQuery(tableName, filters)

	query = s.applyOrderBy(query, orderBy)

	// ??????? CountOptimizer ????????? EXPLAIN ???
	whereClause, whereArgs := s.buildOrderWhereClause(filters)
	countOptimizer := utils.NewCountOptimizer(s.ctx, OrderCountThreshold, "order")
	total, _, err := countOptimizer.OptimizedCountWithTable(tableName, whereClause, whereArgs...)
	if err != nil {
		return nil, 0, err
	}

	// ??????????????????
	findQuery := s.buildShardingQuery(tableName, filters)

	findQuery = s.applyOrderBy(findQuery, orderBy)

	// ??????
	var orders []models.Order
	if err := findQuery.Offset((page - 1) * pageSize).Limit(pageSize).Find(&orders); err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

// GetOrdersWithDetails ?????????????
// ???????????????????? order_no ?? DB ???????????????
func (s *OrderServiceImpl) GetOrdersWithDetails(filters OrderFilters, page, pageSize int) ([]OrderWithDetails, int64, error) {
	if searchorders.QueryEnabled() {
		rows, total, err := s.getOrdersWithDetailsFromSearch(filters, page, pageSize)
		if err == nil {
			return rows, total, nil
		}
		// ?????????????????????????????
		errorlog.Record(s.ctx, "order", "???????????????", map[string]any{
			"driver": search.Driver(),
			"error":  err.Error(),
		}, "??????????: %v", err)
	}

	return s.getOrdersWithDetailsFromDB(filters, page, pageSize)
}

func (s *OrderServiceImpl) getOrdersWithDetailsFromDB(filters OrderFilters, page, pageSize int) ([]OrderWithDetails, int64, error) {
	// ???????
	orderList, total, err := s.GetOrders(filters, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	if len(orderList) == 0 {
		return []OrderWithDetails{}, total, nil
	}

	result := make([]OrderWithDetails, len(orderList))

	// ???????ID
	orderIDsByTable := make(map[string][]uint)
	orderIndexByID := make(map[uint]int)

	for i, order := range orderList {
		// ?????? OrderWithDetails
		result[i] = OrderWithDetails{
			Order:   order,
			Details: []models.OrderDetail{},
		}

		// ????? created_at ??????
		timeStr := order.CreatedAt.ToDateTimeString()
		createdAt, _ := utils.ParseDateTimeUTC(timeStr)

		// ???????
		detailTableName := utils.GetShardingTableName("order_details", createdAt)

		// ???????ID
		if orderIDsByTable[detailTableName] == nil {
			orderIDsByTable[detailTableName] = []uint{}
		}
		orderIDsByTable[detailTableName] = append(orderIDsByTable[detailTableName], order.ID)
		orderIndexByID[order.ID] = i
	}

	// ???????????
	for tableName, orderIDs := range orderIDsByTable {
		if len(orderIDs) == 0 {
			continue
		}
		orderIDsAny := make([]any, len(orderIDs))
		for i, id := range orderIDs {
			orderIDsAny[i] = id
		}

		var details []models.OrderDetail
		if err := appfacades.OrmQuery(s.ctx).Table(tableName).
			WhereIn("order_id", orderIDsAny).
			Find(&details); err == nil {
			// ???????????
			for _, detail := range details {
				if index, ok := orderIndexByID[detail.OrderID]; ok {
					result[index].Details = append(result[index].Details, detail)
				}
			}
		}
	}

	return result, total, nil
}

// applyOrderBy ????
func (s *OrderServiceImpl) applyOrderBy(query orm.Query, orderBy string) orm.Query {
	// ????????????:asc/desc
	parts := strings.Split(orderBy, ":")
	if len(parts) != 2 {
		// ????
		return query.Order("created_at desc")
	}

	field := parts[0]
	direction := strings.ToLower(parts[1])

	// ???????
	allowedFields := map[string]bool{
		"id":         true,
		"order_no":   true,
		"user_id":    true,
		"amount":     true,
		"status":     true,
		"created_at": true,
		"updated_at": true,
	}

	if !allowedFields[field] {
		// ??????????????
		return query.Order("created_at desc")
	}

	if direction == "asc" {
		return query.Order(field + " asc")
	} else {
		return query.Order(field + " desc")
	}
}

// GetAllOrdersForExport ????????????????3???????
// ?? UNION ALL ??????????????
func (s *OrderServiceImpl) GetAllOrdersForExport(filters OrderFilters) ([]models.Order, error) {
	// ?????????3??
	valid, err := utils.ValidateTimeRange(filters.StartTime, filters.EndTime)
	if !valid {
		return nil, err
	}

	// ???????????
	tableNames := utils.GetShardingTableNames("orders", filters.StartTime, filters.EndTime)
	if len(tableNames) == 0 {
		return []models.Order{}, nil
	}

	// ?????????????
	if len(tableNames) == 1 {
		// ????????????????????????
		if !utils.ShardingTableExistsCtx(s.ctx, tableNames[0]) {
			return []models.Order{}, nil
		}

		query := s.buildShardingQuery(tableNames[0], filters)
		orderBy := filters.OrderBy
		if orderBy == "" {
			orderBy = "created_at:desc"
		}
		query = s.applyOrderBy(query, orderBy)

		var orders []models.Order
		if err := query.Find(&orders); err != nil {
			return nil, apperrors.ErrQueryFailed.WithError(err)
		}
		return orders, nil
	}

	// ??????? UNION ALL ????????
	return s.queryMultipleTablesWithUnionForExport(tableNames, filters)
}

// GetAllOrdersWithDetailsForExport ???????????????????3???????
// ?? UNION ALL ??????????????
// ??????????? N+1 ????
func (s *OrderServiceImpl) GetAllOrdersWithDetailsForExport(filters OrderFilters) ([]OrderWithDetails, error) {
	// ????????????? UNION ALL ???
	allOrders, err := s.GetAllOrdersForExport(filters)
	if err != nil {
		return nil, err
	}

	if len(allOrders) == 0 {
		return []OrderWithDetails{}, nil
	}

	// ?????
	result := make([]OrderWithDetails, len(allOrders))

	// ???????ID??? N+1 ??
	orderIDsByTable := make(map[string][]uint)
	orderIndexByID := make(map[uint]int)

	for i, order := range allOrders {
		result[i] = OrderWithDetails{
			Order:   order,
			Details: []models.OrderDetail{},
		}

		// ????? created_at ??????
		timeStr := order.CreatedAt.ToDateTimeString()
		createdAt, _ := utils.ParseDateTimeUTC(timeStr)

		// ???????
		detailTableName := utils.GetShardingTableName("order_details", createdAt)

		// ???????ID
		if orderIDsByTable[detailTableName] == nil {
			orderIDsByTable[detailTableName] = []uint{}
		}
		orderIDsByTable[detailTableName] = append(orderIDsByTable[detailTableName], order.ID)
		orderIndexByID[order.ID] = i
	}

	// ?????????????? N+1 ???
	for tableName, orderIDs := range orderIDsByTable {
		if len(orderIDs) == 0 {
			continue
		}

		// ? []uint ??? []any
		orderIDsAny := make([]any, len(orderIDs))
		for i, id := range orderIDs {
			orderIDsAny[i] = id
		}

		var details []models.OrderDetail
		if err := appfacades.OrmQuery(s.ctx).Table(tableName).
			WhereIn("order_id", orderIDsAny).
			Find(&details); err == nil {
			// ???????????
			for _, detail := range details {
				if index, ok := orderIndexByID[detail.OrderID]; ok {
					result[index].Details = append(result[index].Details, detail)
				}
			}
		}
	}

	return result, nil
}

// UpdateOrder ????????????????????? order_no ????????
func (s *OrderServiceImpl) UpdateOrder(orderID uint, orderTime time.Time, status string, remark string, orderNo ...string) error {
	no := ""
	if len(orderNo) > 0 {
		no = strings.TrimSpace(orderNo[0])
	}
	if no == "" {
		return apperrors.ErrOrderNoRequired
	}
	order, err := s.findOrderByID(orderID, no)
	if err != nil {
		return err
	}

	// ????? created_at ????
	timeStr := order.CreatedAt.ToDateTimeString()
	createdAt, _ := utils.ParseDateTimeUTC(timeStr)
	tableName := utils.GetShardingTableName("orders", createdAt)

	// ??????????????????????
	updateData := map[string]any{
		"status": status,
		"remark": remark,
	}

	_, err = appfacades.OrmQuery(s.ctx).Table(tableName).
		Where("id", orderID).
		Update(updateData)
	if err != nil {
		return err
	}
	support.RequestOrderSearchSync(orderID, order.OrderNo, "index", orderTenantID(s.ctx))
	return nil
}

// DeleteOrder ??????????????????? order_no ????????
func (s *OrderServiceImpl) DeleteOrder(orderID uint, orderTime time.Time, orderNo ...string) error {
	no := ""
	if len(orderNo) > 0 {
		no = strings.TrimSpace(orderNo[0])
	}
	if no == "" {
		return apperrors.ErrOrderNoRequired
	}
	order, err := s.findOrderByID(orderID, no)
	if err != nil {
		return err
	}

	// ????? created_at ?????? carbon.DateTime ??? time.Time?
	// ????????????????
	timeStr := order.CreatedAt.ToDateTimeString()
	createdAt, _ := utils.ParseDateTimeUTC(timeStr)

	// ???????
	detailTableName := utils.GetShardingTableName("order_details", createdAt)
	if _, err := appfacades.OrmQuery(s.ctx).Table(detailTableName).Where("order_id", orderID).Delete(&models.OrderDetail{}); err != nil {
		errorlog.Record(s.ctx, "order", "????????", map[string]any{
			"order_id": orderID,
			"error":    err.Error(),
		}, "????????: %v", err)
		return apperrors.ErrDeleteOrderDetailFailed.WithError(err)
	}

	// ???????
	tableName := utils.GetShardingTableName("orders", createdAt)
	_, err = appfacades.OrmQuery(s.ctx).Table(tableName).Where("id", orderID).Delete(&models.Order{})
	if err != nil {
		return err
	}
	support.RequestOrderSearchSync(orderID, order.OrderNo, "delete", orderTenantID(s.ctx))
	return nil
}

// UpdateOrderByOrderNo ????????????????
func (s *OrderServiceImpl) UpdateOrderByOrderNo(orderNo string, status string, remark string) error {
	// ?????????
	order, err := s.findOrderByOrderNo(orderNo)
	if err != nil {
		return err
	}

	// ????? created_at ????
	timeStr := order.CreatedAt.ToDateTimeString()
	createdAt, _ := utils.ParseDateTimeUTC(timeStr)
	tableName := utils.GetShardingTableName("orders", createdAt)

	// ??????????????????????
	updateData := map[string]any{
		"status": status,
		"remark": remark,
	}

	// ????
	_, err = appfacades.OrmQuery(s.ctx).Table(tableName).Where("order_no", orderNo).Update(updateData)
	if err != nil {
		return err
	}
	support.RequestOrderSearchSync(order.ID, orderNo, "index", orderTenantID(s.ctx))
	return nil
}

// DeleteOrderByOrderNo ??????????????
func (s *OrderServiceImpl) DeleteOrderByOrderNo(orderNo string) error {
	// ?????????
	order, err := s.findOrderByOrderNo(orderNo)
	if err != nil {
		return err
	}

	// ????? created_at ????
	timeStr := order.CreatedAt.ToDateTimeString()
	createdAt, _ := utils.ParseDateTimeUTC(timeStr)

	// ???????
	detailTableName := utils.GetShardingTableName("order_details", createdAt)
	_, err = appfacades.OrmQuery(s.ctx).Table(detailTableName).Where("order_id", order.ID).Delete(&models.OrderDetail{})
	if err != nil {
		errorlog.Record(s.ctx, "order", "????????", map[string]any{
			"order_no": orderNo,
			"error":    err.Error(),
		}, "????????: %v", err)
		return apperrors.ErrDeleteOrderDetailFailed.WithError(err)
	}

	// ???????
	tableName := utils.GetShardingTableName("orders", createdAt)
	_, err = appfacades.OrmQuery(s.ctx).Table(tableName).Where("order_no", orderNo).Delete(&models.Order{})
	if err != nil {
		return err
	}
	support.RequestOrderSearchSync(order.ID, orderNo, "delete", orderTenantID(s.ctx))
	return nil
}

// generateOrderNo ?????????ORD + YYYYMM + ULID?
// ???ORD20250101ARZ3S0K5M2X9P4Q6R8T1V3W5Y7Z9
// ?? 202501 ?? 2025?01??????????
func (s *OrderServiceImpl) generateOrderNo() string {
	return utils.GenerateShardingNo(utils.OrderNoConfig)
}

// findOrderByOrderNo ?? orderrepo?
func (s *OrderServiceImpl) findOrderByOrderNo(orderNo string) (*models.Order, error) {
	return orderrepo.FindOrderByOrderNo(s.ctx, orderNo)
}

// GetOrdersCountInYear ????????????????????
// ?? EXPLAIN ?????????????????????
func (s *OrderServiceImpl) GetOrdersCountInYear() (int64, error) {
	// ???????????
	now := time.Now().UTC()
	startTime := now.AddDate(-1, 0, 0) // ???
	endTime := now

	// ???????????
	tableNames := utils.GetShardingTableNames("orders", startTime, endTime)
	if len(tableNames) == 0 {
		return 0, nil
	}

	var total int64

	// ?? EXPLAIN ???????? COUNT ????
	for _, tableName := range tableNames {
		// ???????
		if !utils.ShardingTableExistsCtx(s.ctx, tableName) {
			continue
		}

		// ?? EXPLAIN ??????
		var explainResult []struct {
			Rows int64 `gorm:"column:rows"`
		}
		sql := fmt.Sprintf("EXPLAIN SELECT * FROM `%s` WHERE created_at >= ? AND created_at <= ?", tableName)
		err := appfacades.OrmQuery(s.ctx).Raw(sql, startTime, endTime).Scan(&explainResult)
		if err != nil {
			errorlog.Record(s.ctx, "order", "??????????", map[string]any{
				"table_name": tableName,
				"error":      err.Error(),
			}, "???? %s ??????: %v", tableName, err)
			continue
		}

		if len(explainResult) > 0 {
			total += explainResult[0].Rows
		}
	}

	return total, nil
}

func (s *OrderServiceImpl) OrderToJSON(order models.Order) map[string]any {
	return OrderToJSONMap(order)
}

func (s *OrderServiceImpl) OrderDetailToJSON(detail models.OrderDetail) map[string]any {
	return OrderDetailToJSONMap(detail)
}

func (s *OrderServiceImpl) OrderWithDetailsToJSON(item *OrderWithDetails) map[string]any {
	return OrderWithDetailsToJSONMap(item)
}
