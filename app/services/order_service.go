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
	// CreateOrder 创建订单（带防重复提交）
	CreateOrder(userID uint, amount float64, products []OrderProduct, requestID string, remark string) (*models.Order, []models.OrderDetail, error)
	// GetOrderByID 根据ID查询订单
	GetOrderByID(orderID uint, orderTime time.Time) (*models.Order, []models.OrderDetail, error)
	// GetOrderByOrderNo 根据订单号查询订单（直接定位分表，更高效）
	GetOrderByOrderNo(orderNo string) (*models.Order, []models.OrderDetail, error)
	// GetOrders 查询订单列表（限制不超过3个月）
	GetOrders(filters OrderFilters, page, pageSize int) ([]models.Order, int64, error)
	// GetOrdersWithDetails 查询订单列表（包含详情，限制不超过3个月）
	GetOrdersWithDetails(filters OrderFilters, page, pageSize int) ([]OrderWithDetails, int64, error)
	// OrderToJSON 订单基础展示字段
	OrderToJSON(order models.Order) map[string]any
	// OrderDetailToJSON 订单明细展示字段
	OrderDetailToJSON(detail models.OrderDetail) map[string]any
	// OrderWithDetailsToJSON 订单列表项（含详情）
	OrderWithDetailsToJSON(item *OrderWithDetails) map[string]any
	// GetAllOrdersForExport 获取所有订单用于导出（限制不超过3个月，不分页）
	GetAllOrdersForExport(filters OrderFilters) ([]models.Order, error)
	// GetAllOrdersWithDetailsForExport 获取所有订单及详情用于导出（限制不超过3个月，不分页）
	GetAllOrdersWithDetailsForExport(filters OrderFilters) ([]OrderWithDetails, error)
	// UpdateOrder 更新订单（必须提供 order_no 以定位分表）
	UpdateOrder(orderID uint, orderTime time.Time, status string, remark string, orderNo ...string) error
	// UpdateOrderByOrderNo 根据订单号更新订单（状态和备注）
	UpdateOrderByOrderNo(orderNo string, status string, remark string) error
	// DeleteOrder 删除订单（必须提供 order_no 以定位分表）
	DeleteOrder(orderID uint, orderTime time.Time, orderNo ...string) error
	// DeleteOrderByOrderNo 根据订单号删除订单
	DeleteOrderByOrderNo(orderNo string) error
	// GetOrdersCountInYear 获取最近一年的订单总数（用于仪表盘统计）
	GetOrdersCountInYear() (int64, error)
	// SearchMyOrdersForUser C 端「我的订单」检索：开启搜索引擎时走索引，否则走分表数据库（关键词仅匹配订单号、备注）。
	SearchMyOrdersForUser(ctx context.Context, userID uint, keyword string, page, pageSize int, tr searchorders.CreatedRange) ([]searchorders.ListItem, int64, error)
}

type OrderServiceImpl struct {
	ctx                  context.Context
	shardingService      ShardingService
	shardingQueryService ShardingQueryService
	countThreshold       int64 // count 查询优化阈值
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

	// 初始化分表查询服务
	service.shardingQueryService = NewShardingQueryService(ctx, ShardingQueryConfig{
		BaseTableName: "orders",
		GetColumns: func() string {
			return service.getOrderTableColumns()
		},
		UnionCollation: "utf8mb4_unicode_ci", // 解决 MySQL UNION 时 "Illegal mix of collations" 错误
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
		CountThreshold: OrderCountThreshold, // 订单数据量大，超过此值使用估算值，不设置或者0直接使用count
	})

	return service
}

// CreateOrder 创建订单（主单 + 明细同事务；DDL 保表在事务外）。
func (s *OrderServiceImpl) CreateOrder(userID uint, amount float64, products []OrderProduct, requestID string, remark string) (*models.Order, []models.OrderDetail, error) {
	if requestID == "" {
		requestID = ulid.Make().String()
	}

	lockKey := tenancy.CacheKey(s.ctx, fmt.Sprintf("order:lock:%s", requestID))
	lockValue := fmt.Sprintf("%d_%d", userID, time.Now().Unix())

	var cachedValue string
	cacheResult := facades.Cache().Get(lockKey, &cachedValue)
	if cacheResult != nil && cachedValue != "" {
		return nil, nil, errors.New("订单正在处理中，请勿重复提交")
	}

	if err := facades.Cache().Put(lockKey, lockValue, 5*time.Second); err != nil {
		errorlog.Record(s.ctx, "order", "获取锁失败", map[string]any{
			"user_id":    userID,
			"request_id": requestID,
			"lock_key":   lockKey,
			"error":      err.Error(),
		}, "获取锁失败: %v", err)
		return nil, nil, apperrors.ErrGetLockFailed.WithError(err)
	}
	defer func() {
		_ = facades.Cache().Forget(lockKey)
	}()

	now := time.Now().UTC()
	tableName := utils.GetShardingTableName("orders", now)
	detailTableName := utils.GetShardingTableName("order_details", now)

	// MySQL DDL 会隐式提交，必须放在事务外。
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
				Status:  "pending",
				Remark:  remark,
			}
			if err := tx.Table(tableName).Create(candidate); err != nil {
				errStr := err.Error()
				if strings.Contains(errStr, "Duplicate entry") || strings.Contains(errStr, "1062") {
					if i == maxRetries-1 {
						errorlog.Record(s.ctx, "order", "生成唯一订单号失败", map[string]any{
							"user_id":    userID,
							"request_id": requestID,
							"retries":    maxRetries,
						}, "生成唯一订单号失败，请重试")
						return apperrors.ErrGenerateOrderNoFailed
					}
					continue
				}
				errorlog.Record(s.ctx, "order", "创建订单失败", map[string]any{
					"user_id":    userID,
					"request_id": requestID,
					"amount":     amount,
					"error":      err.Error(),
				}, "创建订单失败: %v", err)
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
				errorlog.Record(s.ctx, "order", "创建订单详情失败", map[string]any{
					"order_id":   order.ID,
					"user_id":    userID,
					"product_id": product.ProductID,
					"error":      err.Error(),
				}, "创建订单详情失败: %v", err)
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

// findOrderByID 委托 orderrepo，与 ES 同步等只读场景共用同一套分表查找逻辑。
func (s *OrderServiceImpl) findOrderByID(orderID uint, orderNo ...string) (*models.Order, error) {
	if len(orderNo) > 0 && orderNo[0] != "" {
		return orderrepo.FindOrderByID(s.ctx, orderID, orderNo[0])
	}
	return orderrepo.FindOrderByID(s.ctx, orderID)
}

// GetOrderByID 根据ID查询订单
func (s *OrderServiceImpl) GetOrderByID(orderID uint, orderTime time.Time) (*models.Order, []models.OrderDetail, error) {
	return orderrepo.FindOrderWithDetails(s.ctx, orderID, "")
}

// GetOrderByOrderNo 根据订单号查询订单（直接定位分表，更高效）
func (s *OrderServiceImpl) GetOrderByOrderNo(orderNo string) (*models.Order, []models.OrderDetail, error) {
	return orderrepo.FindOrderWithDetails(s.ctx, 0, orderNo)
}

// GetOrders 查询订单列表（限制时间跨度；深分页受 max_union_limit_per_table 约束）
func (s *OrderServiceImpl) GetOrders(filters OrderFilters, page, pageSize int) ([]models.Order, int64, error) {
	// 验证时间范围不超过配置月数
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

	// 获取需要查询的所有分表
	tableNames := utils.GetShardingTableNames("orders", filters.StartTime, filters.EndTime)
	if len(tableNames) == 0 {
		return []models.Order{}, 0, nil
	}

	// 如果只有一个分表，直接查询
	if len(tableNames) == 1 {
		return s.querySingleTable(tableNames[0], filters, page, pageSize)
	}

	// 多个分表：使用 UNION ALL 在数据库层面合并
	return s.queryMultipleTablesWithUnion(tableNames, filters, page, pageSize)
}

// buildShardingQuery 构建分表查询条件（辅助函数，减少重复代码）
func (s *OrderServiceImpl) buildShardingQuery(tableName string, filters OrderFilters) orm.Query {
	return BuildOrderQuery(s.ctx, tableName, filters)
}

// buildOrderWhereClause 构建订单查询的 WHERE 条件（用于通用分表查询服务）
func (s *OrderServiceImpl) buildOrderWhereClause(filters any) (string, []any) {
	orderFilters, ok := filters.(OrderFilters)
	if !ok {
		return "", nil
	}

	var conditions []string
	var args []any

	// 时间范围（必填）
	if !orderFilters.StartTime.IsZero() {
		conditions = append(conditions, "created_at >= ?")
		args = append(args, orderFilters.StartTime)
	}
	if !orderFilters.EndTime.IsZero() {
		conditions = append(conditions, "created_at <= ?")
		args = append(args, orderFilters.EndTime)
	}

	// 用户ID筛选
	if orderFilters.UserID > 0 {
		conditions = append(conditions, "user_id = ?")
		args = append(args, orderFilters.UserID)
	}

	// 订单号模糊搜索
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

	// 订单状态筛选
	if orderFilters.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, orderFilters.Status)
	}

	// 金额范围筛选
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

// getOrderTableColumns 获取订单表的所有列名（用于 UNION ALL 查询）
// 明确指定列名，避免不同分表列数不一致的问题
// 只查询 Order 模型中存在的字段，忽略可能不存在的扩展字段（如 payment_method）
func (s *OrderServiceImpl) getOrderTableColumns() string {
	// 列顺序必须与 Order 模型字段顺序一致
	// 只包含 Order 模型中定义的字段，确保所有分表都有这些字段
	// 注意：不包含 payment_method，因为：
	// 1. Order 模型中没有该字段
	// 2. 某些旧分表可能没有该字段
	// 3. 如果将来需要该字段，应该先更新 Order 模型，然后确保所有分表都有该字段
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

// queryMultipleTablesWithUnion 使用 UNION ALL 查询多个分表
// 在数据库层面合并多个分表，统一排序和分页，性能更优
// 使用通用分表查询服务
func (s *OrderServiceImpl) queryMultipleTablesWithUnion(tableNames []string, filters OrderFilters, page, pageSize int) ([]models.Order, int64, error) {
	var orders []models.Order
	total, err := s.shardingQueryService.QueryMultipleTables(tableNames, filters, page, pageSize, &orders)
	if err != nil {
		return nil, 0, err
	}
	return orders, total, nil
}

// queryMultipleTablesWithUnionForExport 使用 UNION ALL 查询多个分表（用于导出，不分页）
// 在数据库层面合并多个分表，统一排序，性能更优
// 使用通用分表查询服务
func (s *OrderServiceImpl) queryMultipleTablesWithUnionForExport(tableNames []string, filters OrderFilters) ([]models.Order, error) {
	var orders []models.Order
	err := s.shardingQueryService.QueryMultipleTablesForExport(tableNames, filters, &orders)
	if err != nil {
		return nil, err
	}
	return orders, nil
}

// sortOrders 对订单列表进行排序
func (s *OrderServiceImpl) sortOrders(orders []models.Order, orderBy string) {
	parts := strings.Split(orderBy, ":")
	field := "created_at"
	desc := true
	if len(parts) == 2 {
		field = parts[0]
		desc = strings.ToLower(parts[1]) == "desc"
	}

	// 使用 sort.Slice 进行排序
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

// querySingleTable 查询单个分表
func (s *OrderServiceImpl) querySingleTable(tableName string, filters OrderFilters, page, pageSize int) ([]models.Order, int64, error) {
	// 友好处理：目标分表不存在时返回空结果，而不是抛出 SQL 1146 错误。
	if !utils.ShardingTableExistsCtx(s.ctx, tableName) {
		return []models.Order{}, 0, nil
	}

	// 应用排序
	orderBy := filters.OrderBy
	if orderBy == "" {
		orderBy = "created_at:desc"
	}

	// 构建基础查询条件
	query := s.buildShardingQuery(tableName, filters)

	query = s.applyOrderBy(query, orderBy)

	// 获取总数（使用 CountOptimizer 优化，超过阈值使用 EXPLAIN 估算）
	whereClause, whereArgs := s.buildOrderWhereClause(filters)
	countOptimizer := utils.NewCountOptimizer(s.ctx, OrderCountThreshold, "order")
	total, _, err := countOptimizer.OptimizedCountWithTable(tableName, whereClause, whereArgs...)
	if err != nil {
		return nil, 0, err
	}

	// 构建分页查询（重新构建确保使用分表）
	findQuery := s.buildShardingQuery(tableName, filters)

	findQuery = s.applyOrderBy(findQuery, orderBy)

	// 执行分页查询
	var orders []models.Order
	if err := findQuery.Offset((page - 1) * pageSize).Limit(pageSize).Find(&orders); err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

// GetOrdersWithDetails 查询订单列表（包含详情）。
// 当当前搜索驱动检索可用时优先走引擎，再按 order_no 回源 DB 加载明细；失败则回退分表查询。
func (s *OrderServiceImpl) GetOrdersWithDetails(filters OrderFilters, page, pageSize int) ([]OrderWithDetails, int64, error) {
	if searchorders.QueryEnabled() {
		rows, total, err := s.getOrdersWithDetailsFromSearch(filters, page, pageSize)
		if err == nil {
			return rows, total, nil
		}
		// 驱动未实现或临时故障：回退分表，避免把引擎专属错误抛给前端
		errorlog.Record(s.ctx, "order", "搜索引擎列表失败，回退分表查询", map[string]any{
			"driver": search.Driver(),
			"error":  err.Error(),
		}, "搜索引擎订单列表失败: %v", err)
	}

	return s.getOrdersWithDetailsFromDB(filters, page, pageSize)
}

func (s *OrderServiceImpl) getOrdersWithDetailsFromDB(filters OrderFilters, page, pageSize int) ([]OrderWithDetails, int64, error) {
	// 先查询订单列表
	orders, total, err := s.GetOrders(filters, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	if len(orders) == 0 {
		return []OrderWithDetails{}, total, nil
	}

	result := make([]OrderWithDetails, len(orders))

	// 按分表分组订单ID
	orderIDsByTable := make(map[string][]uint)
	orderIndexByID := make(map[uint]int)

	for i, order := range orders {
		// 将订单转换为 OrderWithDetails
		result[i] = OrderWithDetails{
			Order:   order,
			Details: []models.OrderDetail{},
		}

		// 根据订单的 created_at 确定详情分表
		timeStr := order.CreatedAt.ToDateTimeString()
		createdAt, _ := utils.ParseDateTimeUTC(timeStr)

		// 获取详情分表名
		detailTableName := utils.GetShardingTableName("order_details", createdAt)

		// 按分表分组订单ID
		if orderIDsByTable[detailTableName] == nil {
			orderIDsByTable[detailTableName] = []uint{}
		}
		orderIDsByTable[detailTableName] = append(orderIDsByTable[detailTableName], order.ID)
		orderIndexByID[order.ID] = i
	}

	// 按分表批量查询订单详情
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
			// 将详情分配到对应的订单
			for _, detail := range details {
				if index, ok := orderIndexByID[detail.OrderID]; ok {
					result[index].Details = append(result[index].Details, detail)
				}
			}
		}
	}

	return result, total, nil
}

// applyOrderBy 应用排序
func (s *OrderServiceImpl) applyOrderBy(query orm.Query, orderBy string) orm.Query {
	// 解析排序字段，格式：字段:asc/desc
	parts := strings.Split(orderBy, ":")
	if len(parts) != 2 {
		// 默认排序
		return query.Order("created_at desc")
	}

	field := parts[0]
	direction := strings.ToLower(parts[1])

	// 允许排序的字段
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
		// 如果字段不允许，使用默认排序
		return query.Order("created_at desc")
	}

	if direction == "asc" {
		return query.Order(field + " asc")
	} else {
		return query.Order(field + " desc")
	}
}

// GetAllOrdersForExport 获取所有订单用于导出（限制不超过3个月，不分页）
// 使用 UNION ALL 优化，在数据库层面合并和排序
func (s *OrderServiceImpl) GetAllOrdersForExport(filters OrderFilters) ([]models.Order, error) {
	// 验证时间范围不超过3个月
	valid, err := utils.ValidateTimeRange(filters.StartTime, filters.EndTime)
	if !valid {
		return nil, err
	}

	// 获取需要查询的所有分表
	tableNames := utils.GetShardingTableNames("orders", filters.StartTime, filters.EndTime)
	if len(tableNames) == 0 {
		return []models.Order{}, nil
	}

	// 如果只有一个分表，直接查询
	if len(tableNames) == 1 {
		// 友好处理：单表导出时分表不存在，直接返回空数据。
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

	// 多个分表：使用 UNION ALL 在数据库层面合并
	return s.queryMultipleTablesWithUnionForExport(tableNames, filters)
}

// GetAllOrdersWithDetailsForExport 获取所有订单及详情用于导出（限制不超过3个月，不分页）
// 使用 UNION ALL 优化，在数据库层面合并和排序
// 优化：使用批量查询避免 N+1 查询问题
func (s *OrderServiceImpl) GetAllOrdersWithDetailsForExport(filters OrderFilters) ([]OrderWithDetails, error) {
	// 先获取所有订单（使用优化的 UNION ALL 方法）
	allOrders, err := s.GetAllOrdersForExport(filters)
	if err != nil {
		return nil, err
	}

	if len(allOrders) == 0 {
		return []OrderWithDetails{}, nil
	}

	// 初始化结果
	result := make([]OrderWithDetails, len(allOrders))

	// 按分表分组订单ID，避免 N+1 查询
	orderIDsByTable := make(map[string][]uint)
	orderIndexByID := make(map[uint]int)

	for i, order := range allOrders {
		result[i] = OrderWithDetails{
			Order:   order,
			Details: []models.OrderDetail{},
		}

		// 根据订单的 created_at 确定详情分表
		timeStr := order.CreatedAt.ToDateTimeString()
		createdAt, _ := utils.ParseDateTimeUTC(timeStr)

		// 获取详情分表名
		detailTableName := utils.GetShardingTableName("order_details", createdAt)

		// 按分表分组订单ID
		if orderIDsByTable[detailTableName] == nil {
			orderIDsByTable[detailTableName] = []uint{}
		}
		orderIDsByTable[detailTableName] = append(orderIDsByTable[detailTableName], order.ID)
		orderIndexByID[order.ID] = i
	}

	// 按分表批量查询订单详情（避免 N+1 查询）
	for tableName, orderIDs := range orderIDsByTable {
		if len(orderIDs) == 0 {
			continue
		}

		// 将 []uint 转换为 []any
		orderIDsAny := make([]any, len(orderIDs))
		for i, id := range orderIDs {
			orderIDsAny[i] = id
		}

		var details []models.OrderDetail
		if err := appfacades.OrmQuery(s.ctx).Table(tableName).
			WhereIn("order_id", orderIDsAny).
			Find(&details); err == nil {
			// 将详情分配到对应的订单
			for _, detail := range details {
				if index, ok := orderIndexByID[detail.OrderID]; ok {
					result[index].Details = append(result[index].Details, detail)
				}
			}
		}
	}

	return result, nil
}

// UpdateOrder 更新订单（状态和备注）。跨分表场景必须提供 order_no 以正确定位分表。
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

	// 使用订单的 created_at 确定分表
	timeStr := order.CreatedAt.ToDateTimeString()
	createdAt, _ := utils.ParseDateTimeUTC(timeStr)
	tableName := utils.GetShardingTableName("orders", createdAt)

	// 构建更新数据（始终更新备注，即使为空字符串）
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

// DeleteOrder 删除订单（软删除）。跨分表场景必须提供 order_no 以正确定位分表。
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

	// 使用订单的 created_at 确定分表（将 carbon.DateTime 转换为 time.Time）
	// 通过格式化字符串再解析的方式转换
	timeStr := order.CreatedAt.ToDateTimeString()
	createdAt, _ := utils.ParseDateTimeUTC(timeStr)

	// 软删除订单详情
	detailTableName := utils.GetShardingTableName("order_details", createdAt)
	if _, err := appfacades.OrmQuery(s.ctx).Table(detailTableName).Where("order_id", orderID).Delete(&models.OrderDetail{}); err != nil {
		errorlog.Record(s.ctx, "order", "删除订单详情失败", map[string]any{
			"order_id": orderID,
			"error":    err.Error(),
		}, "删除订单详情失败: %v", err)
		return apperrors.ErrDeleteOrderDetailFailed.WithError(err)
	}

	// 软删除订单主表
	tableName := utils.GetShardingTableName("orders", createdAt)
	_, err = appfacades.OrmQuery(s.ctx).Table(tableName).Where("id", orderID).Delete(&models.Order{})
	if err != nil {
		return err
	}
	support.RequestOrderSearchSync(orderID, order.OrderNo, "delete", orderTenantID(s.ctx))
	return nil
}

// UpdateOrderByOrderNo 根据订单号更新订单（状态和备注）
func (s *OrderServiceImpl) UpdateOrderByOrderNo(orderNo string, status string, remark string) error {
	// 通过订单号查找订单
	order, err := s.findOrderByOrderNo(orderNo)
	if err != nil {
		return err
	}

	// 使用订单的 created_at 确定分表
	timeStr := order.CreatedAt.ToDateTimeString()
	createdAt, _ := utils.ParseDateTimeUTC(timeStr)
	tableName := utils.GetShardingTableName("orders", createdAt)

	// 构建更新数据（始终更新备注，即使为空字符串）
	updateData := map[string]any{
		"status": status,
		"remark": remark,
	}

	// 更新订单
	_, err = appfacades.OrmQuery(s.ctx).Table(tableName).Where("order_no", orderNo).Update(updateData)
	if err != nil {
		return err
	}
	support.RequestOrderSearchSync(order.ID, orderNo, "index", orderTenantID(s.ctx))
	return nil
}

// DeleteOrderByOrderNo 根据订单号删除订单（软删除）
func (s *OrderServiceImpl) DeleteOrderByOrderNo(orderNo string) error {
	// 通过订单号查找订单
	order, err := s.findOrderByOrderNo(orderNo)
	if err != nil {
		return err
	}

	// 使用订单的 created_at 确定分表
	timeStr := order.CreatedAt.ToDateTimeString()
	createdAt, _ := utils.ParseDateTimeUTC(timeStr)

	// 软删除订单详情
	detailTableName := utils.GetShardingTableName("order_details", createdAt)
	_, err = appfacades.OrmQuery(s.ctx).Table(detailTableName).Where("order_id", order.ID).Delete(&models.OrderDetail{})
	if err != nil {
		errorlog.Record(s.ctx, "order", "删除订单详情失败", map[string]any{
			"order_no": orderNo,
			"error":    err.Error(),
		}, "删除订单详情失败: %v", err)
		return apperrors.ErrDeleteOrderDetailFailed.WithError(err)
	}

	// 软删除订单主表
	tableName := utils.GetShardingTableName("orders", createdAt)
	_, err = appfacades.OrmQuery(s.ctx).Table(tableName).Where("order_no", orderNo).Delete(&models.Order{})
	if err != nil {
		return err
	}
	support.RequestOrderSearchSync(order.ID, orderNo, "delete", orderTenantID(s.ctx))
	return nil
}

// generateOrderNo 生成订单号（格式：ORD + YYYYMM + ULID）
// 例如：ORD20250101ARZ3S0K5M2X9P4Q6R8T1V3W5Y7Z9
// 其中 202501 表示 2025年01月，用于快速定位分表
func (s *OrderServiceImpl) generateOrderNo() string {
	return utils.GenerateShardingNo(utils.OrderNoConfig)
}

// findOrderByOrderNo 委托 orderrepo。
func (s *OrderServiceImpl) findOrderByOrderNo(orderNo string) (*models.Order, error) {
	return orderrepo.FindOrderByOrderNo(s.ctx, orderNo)
}

// GetOrdersCountInYear 获取最近一年的订单总数（用于仪表盘统计）
// 使用 EXPLAIN 获取预估行数，性能更好（牺牲精确度换速度）
func (s *OrderServiceImpl) GetOrdersCountInYear() (int64, error) {
	// 计算最近一年的时间范围
	now := time.Now().UTC()
	startTime := now.AddDate(-1, 0, 0) // 一年前
	endTime := now

	// 获取需要查询的所有分表
	tableNames := utils.GetShardingTableNames("orders", startTime, endTime)
	if len(tableNames) == 0 {
		return 0, nil
	}

	var total int64

	// 使用 EXPLAIN 获取预估行数（比 COUNT 快很多）
	for _, tableName := range tableNames {
		// 检查表是否存在
		if !utils.ShardingTableExistsCtx(s.ctx, tableName) {
			continue
		}

		// 使用 EXPLAIN 获取预估行数
		var explainResult []struct {
			Rows int64 `gorm:"column:rows"`
		}
		sql := fmt.Sprintf("EXPLAIN SELECT * FROM `%s` WHERE created_at >= ? AND created_at <= ?", tableName)
		err := appfacades.OrmQuery(s.ctx).Raw(sql, startTime, endTime).Scan(&explainResult)
		if err != nil {
			errorlog.Record(s.ctx, "order", "查询分表预估行数失败", map[string]any{
				"table_name": tableName,
				"error":      err.Error(),
			}, "查询分表 %s 预估行数失败: %v", tableName, err)
			continue
		}

		if len(explainResult) > 0 {
			total += explainResult[0].Rows
		}
	}

	return total, nil
}
