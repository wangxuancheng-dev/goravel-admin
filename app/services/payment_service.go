package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/goravel/framework/contracts/database/orm"
	"github.com/goravel/framework/contracts/http"
	"github.com/spf13/cast"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/http/helpers"
	"goravel/app/models"
	"goravel/app/utils"
	"goravel/app/utils/errorlog"
)

type PaymentService interface {
	// PaymentToJSON 支付记录列表/详情展示字段
	PaymentToJSON(payment *models.Payment) map[string]any
	// GetPaymentByID 根据ID获取支付记录
	GetPaymentByID(id uint) (*models.Payment, error)
	// GetPaymentByPaymentNo 根据支付单号获取支付记录
	GetPaymentByPaymentNo(paymentNo string) (*models.Payment, error)
	// GetPayments 获取支付记录列表
	GetPayments(filters PaymentFilters, page, pageSize int) ([]models.Payment, int64, error)
	// CreatePayment 创建支付记录
	CreatePayment(orderNo string, paymentMethodID uint, userID uint, amount float64, remark string) (*models.Payment, error)
	// UpdatePaymentStatus 更新支付状态（必须提供 paymentNo 以定位分表）
	UpdatePaymentStatus(paymentID uint, status string, thirdPartyNo string, payTime *time.Time, failReason string, notifyData map[string]any, paymentNo ...string) error
}

// PaymentFilters 支付记录查询过滤器
type PaymentFilters struct {
	PaymentNo       string
	OrderNo         string
	PaymentMethodID uint
	UserID          uint
	Status          string
	StartTime       time.Time
	EndTime         time.Time
	OrderBy         string
}

// ParsePaymentListTimeRange 解析支付列表时间；开始为空默认近 7 天，结束为空默认当前时间。
func ParsePaymentListTimeRange(startTimeStr, endTimeStr string) (time.Time, time.Time, error) {
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
		endTime = time.Now().UTC()
	} else {
		endTime, err = utils.ParseDateTime(endTimeStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid_end_time")
		}
	}

	return startTime, endTime, nil
}

// BuildPaymentFiltersFromHTTP 从 query/body 构建支付列表/导出筛选（含默认时间）。
func BuildPaymentFiltersFromHTTP(ctx http.Context) (PaymentFilters, error) {
	startTime, endTime, err := ParsePaymentListTimeRange(
		helpers.GetTimeInputOrQueryParam(ctx, "start_time"),
		helpers.GetTimeInputOrQueryParam(ctx, "end_time"),
	)
	if err != nil {
		return PaymentFilters{}, err
	}

	return PaymentFilters{
		PaymentNo:       ctx.Request().Input("payment_no", ctx.Request().Query("payment_no", "")),
		OrderNo:         ctx.Request().Input("order_no", ctx.Request().Query("order_no", "")),
		PaymentMethodID: cast.ToUint(ctx.Request().Input("payment_method_id", ctx.Request().Query("payment_method_id", "0"))),
		UserID:          cast.ToUint(ctx.Request().Input("user_id", ctx.Request().Query("user_id", "0"))),
		Status:          ctx.Request().Input("status", ctx.Request().Query("status", "")),
		StartTime:       startTime,
		EndTime:         endTime,
		OrderBy:         ctx.Request().Input("order_by", ctx.Request().Query("order_by", "")),
	}, nil
}

// PaymentCountThreshold 支付记录分页统计优化阈值（超过此值使用执行计划估算）
const PaymentCountThreshold int64 = 100000

// BuildPaymentQuery 构建支付记录分表查询（包含时间范围 + 通用筛选），供列表查询/导出复用
func BuildPaymentQuery(ctx context.Context, tableName string, filters PaymentFilters) orm.Query {
	query := appfacades.OrmQuery(ctx).Table(tableName).Where("deleted_at IS NULL")

	// 时间范围
	if !filters.StartTime.IsZero() {
		query = query.Where("created_at >= ?", utils.FormatDateTime(filters.StartTime))
	}
	if !filters.EndTime.IsZero() {
		query = query.Where("created_at <= ?", utils.FormatDateTime(filters.EndTime))
	}

	// 精确匹配条件
	if filters.PaymentNo != "" {
		query = query.Where("payment_no = ?", filters.PaymentNo)
	}
	if filters.OrderNo != "" {
		query = query.Where("order_no = ?", filters.OrderNo)
	}
	if filters.PaymentMethodID > 0 {
		query = query.Where("payment_method_id", filters.PaymentMethodID)
	}
	if filters.UserID > 0 {
		query = query.Where("user_id", filters.UserID)
	}
	if filters.Status != "" {
		query = query.Where("status", filters.Status)
	}

	return query
}

type PaymentServiceImpl struct {
	ctx                  context.Context
	shardingService      ShardingService
	shardingQueryService ShardingQueryService
}

func NewPaymentService(ctx context.Context) PaymentService {
	service := &PaymentServiceImpl{
		ctx:             ctx,
		shardingService: NewShardingService(ctx),
	}

	// 初始化分表查询服务
	service.shardingQueryService = NewShardingQueryService(ctx, ShardingQueryConfig{
		BaseTableName: "payments",
		GetColumns: func() string {
			return service.getPaymentTableColumns()
		},
		UnionCollation: "utf8mb4_unicode_ci", // 解决 MySQL UNION 时 "Illegal mix of collations" 错误
		StringColumns:  []string{"payment_no", "order_no", "status", "third_party_no", "fail_reason", "notify_data", "remark"},
		BuildWhereClause: func(filters any) (string, []any) {
			return service.buildPaymentShardingWhereClause(filters)
		},
		GetAllowedOrderFields: func() map[string]bool {
			return map[string]bool{
				"id":         true,
				"payment_no": true,
				"order_no":   true,
				"user_id":    true,
				"amount":     true,
				"status":     true,
				"created_at": true,
				"updated_at": true,
			}
		},
		DefaultOrderBy: "created_at:desc",
		ModuleName:     "payment",
		CountThreshold: PaymentCountThreshold,
	})

	return service
}

func (s *PaymentServiceImpl) PaymentToJSON(payment *models.Payment) map[string]any {
	payload := map[string]any{
		"id":                payment.ID,
		"payment_no":        payment.PaymentNo,
		"order_no":          payment.OrderNo,
		"payment_method_id": payment.PaymentMethodID,
		"user_id":           payment.UserID,
		"amount":            payment.Amount,
		"status":            payment.Status,
		"third_party_no":    payment.ThirdPartyNo,
		"pay_time":          utils.FormatDateTimePtr(payment.PayTime),
		"fail_reason":       payment.FailReason,
		"remark":            payment.Remark,
		"created_at":        payment.CreatedAt,
		"updated_at":        payment.UpdatedAt,
	}
	if payment.PaymentMethod.ID > 0 {
		payload["payment_method"] = map[string]any{
			"id":   payment.PaymentMethod.ID,
			"name": payment.PaymentMethod.Name,
			"code": payment.PaymentMethod.Code,
			"type": payment.PaymentMethod.Type,
		}
	}
	return payload
}

// GetPaymentByID 根据ID获取支付记录（支持分表）
func (s *PaymentServiceImpl) GetPaymentByID(id uint) (*models.Payment, error) {
	// 使用分表查找
	payment, err := s.findPaymentByID(id)
	if err != nil {
		return nil, err
	}
	// 手动加载支付方式
	if payment.PaymentMethodID > 0 {
		paymentMethod, err := NewPaymentMethodService(s.ctx).GetPaymentMethodByID(payment.PaymentMethodID)
		if err == nil {
			payment.PaymentMethod = *paymentMethod
		}
	}
	return payment, nil
}

// GetPaymentByPaymentNo 根据支付单号获取支付记录（支持分表，直接定位更高效）
func (s *PaymentServiceImpl) GetPaymentByPaymentNo(paymentNo string) (*models.Payment, error) {
	// 使用分表查找（通过支付单号直接定位分表）
	payment, err := s.findPaymentByPaymentNo(paymentNo)
	if err != nil {
		return nil, err
	}
	// 手动加载支付方式
	if payment.PaymentMethodID > 0 {
		paymentMethod, err := NewPaymentMethodService(s.ctx).GetPaymentMethodByID(payment.PaymentMethodID)
		if err == nil {
			payment.PaymentMethod = *paymentMethod
		}
	}
	return payment, nil
}

// GetPayments 获取支付记录列表（支持分表，限制不超过3个月）
func (s *PaymentServiceImpl) GetPayments(filters PaymentFilters, page, pageSize int) ([]models.Payment, int64, error) {
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
	tableNames := utils.GetShardingTableNames("payments", filters.StartTime, filters.EndTime)
	if len(tableNames) == 0 {
		return []models.Payment{}, 0, nil
	}

	var payments []models.Payment
	var total int64

	// 如果只有一个分表，直接查询
	if len(tableNames) == 1 {
		payments, total, err = s.querySinglePaymentTable(tableNames[0], filters, page, pageSize)
	} else {
		// 多个分表：使用 UNION ALL 在数据库层面合并
		payments, total, err = s.queryMultiplePaymentTablesWithUnion(tableNames, filters, page, pageSize)
	}

	if err != nil {
		return nil, 0, err
	}

	// 批量加载支付方式
	paymentMethodIDs := make(map[uint]bool)
	for _, payment := range payments {
		if payment.PaymentMethodID > 0 {
			paymentMethodIDs[payment.PaymentMethodID] = true
		}
	}

	// 批量查询支付方式
	paymentMethodsMap := make(map[uint]*models.PaymentMethod)
	if len(paymentMethodIDs) > 0 {
		var ids []uint
		for id := range paymentMethodIDs {
			ids = append(ids, id)
		}
		// 转换为 []any
		idsAny := make([]any, len(ids))
		for i, id := range ids {
			idsAny[i] = id
		}
		var paymentMethods []models.PaymentMethod
		if err := appfacades.OrmQuery(s.ctx).Model(&models.PaymentMethod{}).WhereIn("id", idsAny).Find(&paymentMethods); err == nil {
			for i := range paymentMethods {
				paymentMethodsMap[paymentMethods[i].ID] = &paymentMethods[i]
			}
		}
	}

	// 关联支付方式
	for i := range payments {
		if pm, ok := paymentMethodsMap[payments[i].PaymentMethodID]; ok {
			payments[i].PaymentMethod = *pm
		}
	}

	return payments, total, nil
}

// CreatePayment 创建支付记录（写入分表）
func (s *PaymentServiceImpl) CreatePayment(orderNo string, paymentMethodID uint, userID uint, amount float64, remark string) (*models.Payment, error) {
	// 验证金额
	if amount <= 0 {
		return nil, apperrors.ErrPaymentAmountInvalid
	}

	// 验证支付方式
	paymentMethod, err := NewPaymentMethodService(s.ctx).GetPaymentMethodByID(paymentMethodID)
	if err != nil {
		return nil, err
	}
	if !paymentMethod.IsActive {
		return nil, apperrors.ErrPaymentMethodDisabled
	}

	// 当前时间用于分表
	now := time.Now()

	// 确保分表存在
	tableName, err := s.ensurePaymentShardingTableExists(now)
	if err != nil {
		return nil, apperrors.ErrCreatePaymentFailed.WithError(err)
	}

	// 生成支付单号（包含日期，便于后续定位分表）
	paymentNo := utils.GenerateShardingNo(utils.PaymentNoConfig)

	payment := &models.Payment{
		PaymentNo:       paymentNo,
		OrderNo:         orderNo,
		PaymentMethodID: paymentMethodID,
		UserID:          userID,
		Amount:          amount,
		Status:          "pending",
		Remark:          remark,
	}

	// 写入分表
	if err := appfacades.OrmQuery(s.ctx).Table(tableName).Create(payment); err != nil {
		errorlog.Record(s.ctx, "payment", "创建支付记录失败", map[string]any{
			"table_name":        tableName,
			"order_no":          orderNo,
			"payment_method_id": paymentMethodID,
			"user_id":           userID,
			"amount":            amount,
			"error":             err.Error(),
		}, "创建支付记录失败: %v", err)
		return nil, apperrors.ErrCreatePaymentFailed.WithError(err)
	}

	return payment, nil
}

// UpdatePaymentStatus 更新支付状态（支持分表）。必须提供 paymentNo 以正确定位分表。
func (s *PaymentServiceImpl) UpdatePaymentStatus(paymentID uint, status string, thirdPartyNo string, payTime *time.Time, failReason string, notifyData map[string]any, paymentNo ...string) error {
	no := ""
	if len(paymentNo) > 0 {
		no = strings.TrimSpace(paymentNo[0])
	}
	if no == "" {
		return apperrors.ErrPaymentNoRequired
	}

	payment, err := s.findPaymentByPaymentNo(no)
	if err != nil {
		return apperrors.ErrPaymentNotFound.WithError(err)
	}
	if paymentID > 0 && payment.ID != paymentID {
		return apperrors.ErrPaymentNotFound
	}

	// 确定分表名
	timeStr := payment.CreatedAt.ToDateTimeString()
	createdAt, _ := utils.ParseDateTimeUTC(timeStr)
	tableName := utils.GetShardingTableName("payments", createdAt)

	updateData := map[string]any{
		"status": status,
	}

	if thirdPartyNo != "" {
		updateData["third_party_no"] = thirdPartyNo
	}
	if payTime != nil {
		updateData["pay_time"] = payTime
	}
	if failReason != "" {
		updateData["fail_reason"] = failReason
	}
	if notifyData != nil {
		notifyJSON, err := json.Marshal(notifyData)
		if err == nil {
			updateData["notify_data"] = string(notifyJSON)
		}
	}

	if _, err := appfacades.OrmQuery(s.ctx).Table(tableName).Where("id", payment.ID).Update(&models.Payment{}, updateData); err != nil {
		return apperrors.ErrUpdateFailed.WithError(err)
	}

	return nil
}

// buildPaymentWhereClause 构建支付记录查询的 WHERE 条件
func (s *PaymentServiceImpl) buildPaymentWhereClause(filters PaymentFilters) (string, []any) {
	var conditions []string
	var args []any

	if filters.PaymentNo != "" {
		conditions = append(conditions, "payment_no LIKE ?")
		args = append(args, filters.PaymentNo+"%")
	}
	if filters.OrderNo != "" {
		conditions = append(conditions, "order_no LIKE ?")
		args = append(args, filters.OrderNo+"%")
	}
	if filters.PaymentMethodID > 0 {
		conditions = append(conditions, "payment_method_id = ?")
		args = append(args, filters.PaymentMethodID)
	}
	if filters.UserID > 0 {
		conditions = append(conditions, "user_id = ?")
		args = append(args, filters.UserID)
	}
	if filters.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, filters.Status)
	}
	if !filters.StartTime.IsZero() {
		conditions = append(conditions, "created_at >= ?")
		args = append(args, filters.StartTime)
	}
	if !filters.EndTime.IsZero() {
		conditions = append(conditions, "created_at <= ?")
		args = append(args, filters.EndTime)
	}

	if len(conditions) == 0 {
		return "", nil
	}
	return strings.Join(conditions, " AND "), args
}

// applyOrderBy 应用排序
func (s *PaymentServiceImpl) applyOrderBy(query orm.Query, orderBy string) orm.Query {
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
		"name":       true,
		"code":       true,
		"type":       true,
		"is_active":  true,
		"sort":       true,
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

// getPaymentTableColumns 获取支付记录表的所有列名（用于 UNION ALL 查询）
func (s *PaymentServiceImpl) getPaymentTableColumns() string {
	columns := []string{
		"id",
		"payment_no",
		"order_no",
		"payment_method_id",
		"user_id",
		"amount",
		"status",
		"third_party_no",
		"pay_time",
		"fail_reason",
		"notify_data",
		"remark",
		"created_at",
		"updated_at",
		"deleted_at",
	}
	return strings.Join(columns, ", ")
}

// buildPaymentShardingWhereClause 构建支付记录分表查询的 WHERE 条件（用于通用分表查询服务）
func (s *PaymentServiceImpl) buildPaymentShardingWhereClause(filters any) (string, []any) {
	paymentFilters, ok := filters.(PaymentFilters)
	if !ok {
		return "", nil
	}

	var conditions []string
	var args []any

	// 时间范围（必填）
	if !paymentFilters.StartTime.IsZero() {
		conditions = append(conditions, "created_at >= ?")
		args = append(args, paymentFilters.StartTime)
	}
	if !paymentFilters.EndTime.IsZero() {
		conditions = append(conditions, "created_at <= ?")
		args = append(args, paymentFilters.EndTime)
	}

	// 支付单号筛选
	if paymentFilters.PaymentNo != "" {
		conditions = append(conditions, "payment_no = ?")
		args = append(args, paymentFilters.PaymentNo)
	}

	// 订单号筛选
	if paymentFilters.OrderNo != "" {
		conditions = append(conditions, "order_no = ?")
		args = append(args, paymentFilters.OrderNo)
	}

	// 支付方式ID筛选
	if paymentFilters.PaymentMethodID > 0 {
		conditions = append(conditions, "payment_method_id = ?")
		args = append(args, paymentFilters.PaymentMethodID)
	}

	// 用户ID筛选
	if paymentFilters.UserID > 0 {
		conditions = append(conditions, "user_id = ?")
		args = append(args, paymentFilters.UserID)
	}

	// 状态筛选
	if paymentFilters.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, paymentFilters.Status)
	}

	return strings.Join(conditions, " AND "), args
}

// findPaymentByPaymentNo 通过支付单号查找支付记录（直接定位分表）
// 支付单号格式：PAY + YYYYMMDD + ULID，可以从中提取日期
func (s *PaymentServiceImpl) findPaymentByPaymentNo(paymentNo string) (*models.Payment, error) {
	// 从支付单号中解析日期
	if parsedTime, ok := utils.ParseShardingNoDate(paymentNo, utils.PaymentNoConfig); ok {
		// 成功解析日期，直接查询对应分表
		tableName := utils.GetShardingTableName("payments", parsedTime)
		if utils.ShardingTableExistsCtx(s.ctx, tableName) {
			var payment models.Payment
			if err := appfacades.OrmQuery(s.ctx).Model(&models.Payment{}).Table(tableName).Where("payment_no", paymentNo).First(&payment); err == nil {
				return &payment, nil
			}
		}
	}

	// 如果无法从支付单号解析日期，或者在对应分表中找不到，遍历最近 N 个月的分表
	now := time.Now().UTC()
	startTime := now.AddDate(0, -utils.GetIDLookupScanMonths(), 0)
	tableNames := utils.GetShardingTableNames("payments", startTime, now)

	// 从最新的分表开始查询（Model 自动应用软删除过滤）
	for i := len(tableNames) - 1; i >= 0; i-- {
		if !utils.ShardingTableExistsCtx(s.ctx, tableNames[i]) {
			continue
		}
		var payment models.Payment
		if err := appfacades.OrmQuery(s.ctx).Model(&models.Payment{}).Table(tableNames[i]).Where("payment_no", paymentNo).First(&payment); err == nil {
			return &payment, nil
		}
	}

	return nil, apperrors.ErrPaymentNotFound
}

// findPaymentByID 通过支付记录ID查找支付记录
func (s *PaymentServiceImpl) findPaymentByID(paymentID uint, paymentNo ...string) (*models.Payment, error) {
	// 如果提供了支付单号，优先使用支付单号直接定位分表（更高效）
	if len(paymentNo) > 0 && paymentNo[0] != "" {
		payment, err := s.findPaymentByPaymentNo(paymentNo[0])
		if err == nil {
			if paymentID > 0 && payment.ID != paymentID {
				// 支付单号找到了但ID不匹配，继续用ID查找
			} else {
				return payment, nil
			}
		}
	}

	// 如果没有支付单号或通过支付单号查找失败，使用ID遍历分表
	if paymentID == 0 {
		return nil, apperrors.ErrPaymentNotFound
	}

	// 查询最近 N 个月的分表（无单号时的兜底扫描）
	now := time.Now().UTC()
	startTime := now.AddDate(0, -utils.GetIDLookupScanMonths(), 0)
	tableNames := utils.GetShardingTableNames("payments", startTime, now)

	// 从最新的分表开始查询（Model 自动应用软删除过滤）
	for i := len(tableNames) - 1; i >= 0; i-- {
		if !utils.ShardingTableExistsCtx(s.ctx, tableNames[i]) {
			continue
		}
		var payment models.Payment
		if err := appfacades.OrmQuery(s.ctx).Model(&models.Payment{}).Table(tableNames[i]).Where("id", paymentID).First(&payment); err == nil {
			return &payment, nil
		}
	}

	return nil, apperrors.ErrPaymentNotFound
}

// querySinglePaymentTable 查询单个分表
func (s *PaymentServiceImpl) querySinglePaymentTable(tableName string, filters PaymentFilters, page, pageSize int) ([]models.Payment, int64, error) {
	// 友好处理：目标分表不存在时返回空结果，而不是抛出 SQL 1146 错误。
	if !utils.ShardingTableExistsCtx(s.ctx, tableName) {
		return []models.Payment{}, 0, nil
	}

	// 应用排序
	orderBy := filters.OrderBy
	if orderBy == "" {
		orderBy = "created_at:desc"
	}

	// 构建基础查询条件
	query := s.buildPaymentShardingQuery(tableName, filters)
	query = s.applyOrderBy(query, orderBy)

	// 获取总数（使用 CountOptimizer 优化，超过阈值使用 EXPLAIN 估算）
	whereClause, whereArgs := s.buildPaymentShardingWhereClause(filters)
	countOptimizer := utils.NewCountOptimizer(s.ctx, PaymentCountThreshold, "payment")
	total, _, err := countOptimizer.OptimizedCountWithTable(tableName, whereClause, whereArgs...)
	if err != nil {
		return nil, 0, err
	}

	// 执行分页查询
	var payments []models.Payment
	if err := query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&payments); err != nil {
		return nil, 0, err
	}

	return payments, total, nil
}

// buildPaymentShardingQuery 构建分表查询条件
func (s *PaymentServiceImpl) buildPaymentShardingQuery(tableName string, filters PaymentFilters) orm.Query {
	return BuildPaymentQuery(s.ctx, tableName, filters)
}

// queryMultiplePaymentTablesWithUnion 使用 UNION ALL 查询多个分表
func (s *PaymentServiceImpl) queryMultiplePaymentTablesWithUnion(tableNames []string, filters PaymentFilters, page, pageSize int) ([]models.Payment, int64, error) {
	var payments []models.Payment
	total, err := s.shardingQueryService.QueryMultipleTables(tableNames, filters, page, pageSize, &payments)
	if err != nil {
		return nil, 0, err
	}
	return payments, total, nil
}

// ensurePaymentShardingTableExists 确保支付记录分表存在
func (s *PaymentServiceImpl) ensurePaymentShardingTableExists(paymentTime time.Time) (string, error) {
	tableName := utils.GetShardingTableName("payments", paymentTime)
	if err := s.shardingService.EnsureShardingTable(tableName, "payments"); err != nil {
		return "", err
	}
	return tableName, nil
}
