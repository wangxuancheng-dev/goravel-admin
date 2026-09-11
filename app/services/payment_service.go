package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dromara/carbon/v2"
	"github.com/go-pay/gopay"
	"github.com/go-pay/gopay/alipay"
	"github.com/go-pay/gopay/wechat/v3"
	"github.com/goravel/framework/contracts/database/orm"
	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	admin "goravel/app/http/requests/admin"
	"goravel/app/models"
	"goravel/app/utils"
	"goravel/app/utils/errorlog"
	"goravel/database/migrations"
)

type PaymentService interface {
	// GetPaymentMethodByID 根据ID获取支付方式
	GetPaymentMethodByID(id uint) (*models.PaymentMethod, error)
	// GetPaymentMethodByCode 根据代码获取支付方式
	GetPaymentMethodByCode(code string) (*models.PaymentMethod, error)
	// GetPaymentMethods 获取支付方式列表
	GetPaymentMethods(filters PaymentMethodFilters, page, pageSize int) ([]models.PaymentMethod, int64, error)
	// CreatePaymentMethod 创建支付方式
	CreatePaymentMethod(name, code, paymentType string, config map[string]any, isActive bool, sort int, description string) (*models.PaymentMethod, error)
	// CreatePaymentMethodFromRequest 按请求创建支付方式
	CreatePaymentMethodFromRequest(req *admin.PaymentMethodCreate) (*models.PaymentMethod, error)
	// UpdatePaymentMethod 更新支付方式（保留兼容）
	UpdatePaymentMethod(id uint, name string, config map[string]any, isActive bool, sort int, description string) error
	// UpdatePaymentMethodModel 更新支付方式（Save）
	UpdatePaymentMethodModel(paymentMethod *models.PaymentMethod) error
	// UpdatePaymentMethodByRequest 按请求部分更新支付方式
	UpdatePaymentMethodByRequest(httpCtx http.Context, id uint, req *admin.PaymentMethodUpdate) (*models.PaymentMethod, error)
	// DeletePaymentMethod 删除支付方式
	DeletePaymentMethod(id uint) error
	// PaymentMethodListItem 列表行（不含敏感配置）
	PaymentMethodListItem(pm models.PaymentMethod) map[string]any
	// PaymentMethodDetail 详情（含解析后的 config）
	PaymentMethodDetail(pm *models.PaymentMethod) map[string]any
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
	// UpdatePaymentStatus 更新支付状态（paymentNo 可选，提供则更快定位分表）
	UpdatePaymentStatus(paymentID uint, status string, thirdPartyNo string, payTime *time.Time, failReason string, notifyData map[string]any, paymentNo ...string) error

	// CreatePaymentOrder 创建支付订单（调用第三方支付）
	CreatePaymentOrder(payment *models.Payment, clientIP string) (map[string]any, error)
	// QueryPaymentOrder 查询支付订单状态
	QueryPaymentOrder(payment *models.Payment) (map[string]any, error)
	// HandlePaymentNotify 处理支付回调通知
	HandlePaymentNotify(paymentMethod *models.PaymentMethod, notifyData map[string]any) (*models.Payment, error)
}

// PaymentMethodFilters 支付方式查询过滤器
type PaymentMethodFilters struct {
	Name        string
	Code        string
	Type        string
	IsActive    string
	Description string
	OrderBy     string
}

func BuildPaymentMethodFiltersFromHTTP(ctx http.Context) PaymentMethodFilters {
	return PaymentMethodFilters{
		Name:        ctx.Request().Query("name", ""),
		Code:        ctx.Request().Query("code", ""),
		Type:        ctx.Request().Query("type", ""),
		IsActive:    ctx.Request().Query("is_active", ""),
		Description: ctx.Request().Query("description", ""),
		OrderBy:     ctx.Request().Query("order_by", ""),
	}
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

// GetPaymentMethodByID 根据ID获取支付方式
func (s *PaymentServiceImpl) GetPaymentMethodByID(id uint) (*models.PaymentMethod, error) {
	var paymentMethod models.PaymentMethod
	if err := appfacades.OrmQuery(s.ctx).Where("id", id).FirstOrFail(&paymentMethod); err != nil {
		return nil, apperrors.ErrPaymentMethodNotFound.WithError(err)
	}
	return &paymentMethod, nil
}

// GetPaymentMethodByCode 根据代码获取支付方式
func (s *PaymentServiceImpl) GetPaymentMethodByCode(code string) (*models.PaymentMethod, error) {
	var paymentMethod models.PaymentMethod
	if err := appfacades.OrmQuery(s.ctx).Where("code", code).Where("is_active", true).FirstOrFail(&paymentMethod); err != nil {
		return nil, apperrors.ErrPaymentMethodNotFound.WithError(err)
	}
	return &paymentMethod, nil
}

// GetPaymentMethods 获取支付方式列表
func (s *PaymentServiceImpl) GetPaymentMethods(filters PaymentMethodFilters, page, pageSize int) ([]models.PaymentMethod, int64, error) {
	query := appfacades.OrmQuery(s.ctx).Model(&models.PaymentMethod{})

	// 应用筛选条件
	if filters.Name != "" {
		query = query.Where("name LIKE ?", "%"+filters.Name+"%")
	}
	if filters.Code != "" {
		query = query.Where("code", filters.Code)
	}
	if filters.Type != "" {
		query = query.Where("type", filters.Type)
	}
	if filters.IsActive != "" {
		switch filters.IsActive {
		case "1":
			query = query.Where("is_active", true)
		case "0":
			query = query.Where("is_active", false)
		}
	}

	// 应用排序
	if filters.OrderBy != "" {
		query = s.applyOrderBy(query, filters.OrderBy)
	} else {
		query = query.Order("sort asc").Order("id desc")
	}

	// 分页查询
	var paymentMethods []models.PaymentMethod
	var total int64
	if err := query.Paginate(page, pageSize, &paymentMethods, &total); err != nil {
		return nil, 0, apperrors.ErrQueryFailed.WithError(err)
	}

	return paymentMethods, total, nil
}

// validatePaymentMethodCodeUnique 校验支付方式代码唯一性（软删后仍占位）。
func (s *PaymentServiceImpl) validatePaymentMethodCodeUnique(code string, excludeID uint) error {
	if code == "" {
		return nil
	}
	exists, err := utils.ExistsColumnValue(s.ctx, "payment_methods", nil, utils.UniqueReuseDeny, "code", code, excludeID)
	if err != nil {
		return apperrors.ErrCreateFailed.WithError(err)
	}
	if exists {
		return apperrors.ErrPaymentMethodCodeExists
	}
	return nil
}

// CreatePaymentMethod 创建支付方式
func (s *PaymentServiceImpl) CreatePaymentMethod(name, code, paymentType string, config map[string]any, isActive bool, sort int, description string) (*models.PaymentMethod, error) {
	if err := s.validatePaymentMethodCodeUnique(code, 0); err != nil {
		return nil, err
	}
	// 序列化配置
	configJSON, err := json.Marshal(config)
	if err != nil {
		return nil, apperrors.ErrPaymentConfigRequired.WithError(err)
	}

	// 使用 map 创建，确保 IsActive 为 false 时也能正确保存
	// GORM 在处理结构体时会忽略零值字段，使用 map 可以确保所有字段都被保存
	now := carbon.Now()
	paymentMethod := &models.PaymentMethod{}
	createData := map[string]any{
		"name":        name,
		"code":        code,
		"type":        paymentType,
		"config":      string(configJSON),
		"is_active":   isActive,
		"sort":        sort,
		"description": description,
		"created_at":  now,
		"updated_at":  now,
	}

	if err := appfacades.OrmQuery(s.ctx).Model(paymentMethod).Create(createData); err != nil {
		return nil, apperrors.ErrCreateFailed.WithError(err)
	}

	return paymentMethod, nil
}

// CreatePaymentMethodFromRequest 按请求创建支付方式
func (s *PaymentServiceImpl) CreatePaymentMethodFromRequest(req *admin.PaymentMethodCreate) (*models.PaymentMethod, error) {
	return s.CreatePaymentMethod(req.Name, req.Code, req.Type, req.Config, req.IsActive, req.Sort, req.Description)
}

// UpdatePaymentMethod 更新支付方式
func (s *PaymentServiceImpl) UpdatePaymentMethod(id uint, name string, config map[string]any, isActive bool, sort int, description string) error {
	paymentMethod, err := s.GetPaymentMethodByID(id)
	if err != nil {
		return err
	}

	// 序列化配置
	var configJSON string
	if config != nil {
		configBytes, err := json.Marshal(config)
		if err != nil {
			return apperrors.ErrPaymentConfigRequired.WithError(err)
		}
		configJSON = string(configBytes)
	} else {
		configJSON = paymentMethod.Config // 保持原有配置
	}

	updateData := map[string]any{
		"name":        name,
		"config":      configJSON,
		"is_active":   isActive,
		"sort":        sort,
		"description": description,
	}

	if _, err := appfacades.OrmQuery(s.ctx).Where("id", id).Update(&models.PaymentMethod{}, updateData); err != nil {
		return apperrors.ErrUpdateFailed.WithError(err)
	}

	return nil
}

// UpdatePaymentMethodModel 更新支付方式（新模式）
func (s *PaymentServiceImpl) UpdatePaymentMethodModel(paymentMethod *models.PaymentMethod) error {
	if err := appfacades.OrmQuery(s.ctx).Save(paymentMethod); err != nil {
		return apperrors.ErrUpdateFailed.WithError(err)
	}
	return nil
}

// UpdatePaymentMethodByRequest 按请求部分更新支付方式
func (s *PaymentServiceImpl) UpdatePaymentMethodByRequest(httpCtx http.Context, id uint, req *admin.PaymentMethodUpdate) (*models.PaymentMethod, error) {
	paymentMethod, err := s.GetPaymentMethodByID(id)
	if err != nil {
		return nil, err
	}

	allInputs := httpCtx.Request().All()

	if req.Name != nil {
		paymentMethod.Name = *req.Name
	}
	if _, exists := allInputs["config"]; exists {
		if req.Config == nil {
			return nil, apperrors.ErrPaymentConfigRequired
		}
		configBytes, err := json.Marshal(req.Config)
		if err != nil {
			return nil, apperrors.ErrPaymentConfigRequired.WithError(err)
		}
		paymentMethod.Config = string(configBytes)
	}
	if req.IsActive != nil {
		paymentMethod.IsActive = *req.IsActive
	}
	if req.Sort != nil {
		paymentMethod.Sort = *req.Sort
	}
	if req.Description != nil {
		paymentMethod.Description = *req.Description
	}

	if err := s.UpdatePaymentMethodModel(paymentMethod); err != nil {
		return nil, err
	}
	return paymentMethod, nil
}

// DeletePaymentMethod 删除支付方式
func (s *PaymentServiceImpl) DeletePaymentMethod(id uint) error {
	_, err := s.GetPaymentMethodByID(id)
	if err != nil {
		return err
	}

	if _, err := appfacades.OrmQuery(s.ctx).Where("id", id).Delete(&models.PaymentMethod{}); err != nil {
		return apperrors.ErrDeleteFailed.WithError(err)
	}

	return nil
}

func (s *PaymentServiceImpl) PaymentMethodListItem(pm models.PaymentMethod) map[string]any {
	return map[string]any{
		"id":          pm.ID,
		"name":        pm.Name,
		"code":        pm.Code,
		"type":        pm.Type,
		"is_active":   pm.IsActive,
		"sort":        pm.Sort,
		"description": pm.Description,
		"created_at":  pm.CreatedAt,
		"updated_at":  pm.UpdatedAt,
	}
}

func (s *PaymentServiceImpl) PaymentMethodDetail(pm *models.PaymentMethod) map[string]any {
	config := make(map[string]any)
	if pm.Config != "" {
		if err := json.Unmarshal([]byte(pm.Config), &config); err != nil {
			config = make(map[string]any)
		}
	}

	return map[string]any{
		"id":          pm.ID,
		"name":        pm.Name,
		"code":        pm.Code,
		"type":        pm.Type,
		"config":      config,
		"is_active":   pm.IsActive,
		"sort":        pm.Sort,
		"description": pm.Description,
		"created_at":  pm.CreatedAt,
		"updated_at":  pm.UpdatedAt,
	}
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
		paymentMethod, err := s.GetPaymentMethodByID(payment.PaymentMethodID)
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
		paymentMethod, err := s.GetPaymentMethodByID(payment.PaymentMethodID)
		if err == nil {
			payment.PaymentMethod = *paymentMethod
		}
	}
	return payment, nil
}

// GetPayments 获取支付记录列表（支持分表，限制不超过3个月）
func (s *PaymentServiceImpl) GetPayments(filters PaymentFilters, page, pageSize int) ([]models.Payment, int64, error) {
	// 验证时间范围不超过3个月
	valid, err := utils.ValidateTimeRange(filters.StartTime, filters.EndTime)
	if !valid {
		return nil, 0, err
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
		return nil, 0, apperrors.ErrQueryFailed.WithError(err)
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
	paymentMethod, err := s.GetPaymentMethodByID(paymentMethodID)
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

// UpdatePaymentStatus 更新支付状态（支持分表）
// paymentNo 可选，如果提供则可以更快定位分表
func (s *PaymentServiceImpl) UpdatePaymentStatus(paymentID uint, status string, thirdPartyNo string, payTime *time.Time, failReason string, notifyData map[string]any, paymentNo ...string) error {
	// 先查找支付记录以确定分表
	var payment *models.Payment
	var err error
	if len(paymentNo) > 0 && paymentNo[0] != "" {
		payment, err = s.findPaymentByPaymentNo(paymentNo[0])
	} else {
		payment, err = s.findPaymentByID(paymentID)
	}
	if err != nil {
		return apperrors.ErrPaymentNotFound.WithError(err)
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

// CreatePaymentOrder 创建支付订单（调用第三方支付）
func (s *PaymentServiceImpl) CreatePaymentOrder(payment *models.Payment, clientIP string) (map[string]any, error) {
	// 获取支付方式
	paymentMethod, err := s.GetPaymentMethodByID(payment.PaymentMethodID)
	if err != nil {
		return nil, err
	}

	// 解析配置
	var config map[string]any
	if err := json.Unmarshal([]byte(paymentMethod.Config), &config); err != nil {
		return nil, apperrors.ErrPaymentConfigRequired.WithError(err)
	}

	// 根据支付类型调用不同的支付接口
	switch paymentMethod.Type {
	case "wechat":
		return s.createWechatPayment(payment, config, clientIP)
	case "alipay":
		return s.createAlipayPayment(payment, config, clientIP)
	default:
		return nil, apperrors.ErrInvalidPaymentType.WithMessage(fmt.Sprintf("不支持的支付类型: %s", paymentMethod.Type))
	}
}

// createWechatPayment 创建微信支付订单
func (s *PaymentServiceImpl) createWechatPayment(payment *models.Payment, config map[string]any, clientIP string) (map[string]any, error) {
	// 这里需要根据 gopay 的微信支付文档实现
	// 示例代码，实际使用时需要根据 gopay 的最新 API 调整
	// 参考: https://github.com/go-pay/gopay/tree/main/wechat/v3

	// 获取配置参数
	appID, _ := config["app_id"].(string)
	mchID, _ := config["mch_id"].(string)
	apiV3Key, _ := config["api_v3_key"].(string)
	certSerialNo, _ := config["cert_serial_no"].(string)
	privateKeyPath, _ := config["private_key_path"].(string)

	if appID == "" || mchID == "" || apiV3Key == "" {
		return nil, apperrors.ErrPaymentConfigRequired.WithMessage("微信支付配置不完整")
	}

	// 创建微信支付客户端
	client, err := wechat.NewClientV3(appID, mchID, apiV3Key, certSerialNo)
	if err != nil {
		return nil, apperrors.ErrCreatePaymentFailed.WithError(err)
	}
	// 设置私钥（如果需要）
	if privateKeyPath != "" {
		// 这里需要根据 gopay 的实际 API 设置私钥
		// client.SetPrivateKey(...)
	}

	// 设置回调地址（需要从配置中读取）
	notifyURL, _ := config["notify_url"].(string)
	if notifyURL == "" {
		notifyURL = fmt.Sprintf("%s/api/payment/notify/wechat", facades.Config().GetString("app.url"))
	}

	// 创建支付订单
	bm := make(gopay.BodyMap)
	bm.Set("out_trade_no", payment.PaymentNo)
	bm.Set("description", payment.Remark)
	bm.Set("amount", map[string]any{
		"total":    int(payment.Amount * 100), // 转换为分
		"currency": "CNY",
	})
	bm.Set("notify_url", notifyURL)
	bm.Set("payer", map[string]any{
		"openid": config["openid"], // 需要从订单或用户信息中获取
	})

	// 注意：这里需要根据 gopay 的最新 API 调用
	// 示例代码，实际使用时需要根据 gopay 的最新文档调整
	ctx := context.Background()
	wxRsp, err := client.V3TransactionJsapi(ctx, bm)
	if err != nil {
		return nil, apperrors.ErrCreatePaymentFailed.WithError(err)
	}

	if wxRsp.Code != wechat.Success {
		return nil, apperrors.ErrCreatePaymentFailed.WithMessage(wxRsp.Error)
	}

	return map[string]any{
		"payment_no": payment.PaymentNo,
		"prepay_id":  wxRsp.Response.PrepayId,
		// PaySign 可能需要从其他地方获取或计算
	}, nil
}

// createAlipayPayment 创建支付宝支付订单
func (s *PaymentServiceImpl) createAlipayPayment(payment *models.Payment, config map[string]any, clientIP string) (map[string]any, error) {
	// 这里需要根据 gopay 的支付宝文档实现
	// 示例代码，实际使用时需要根据 gopay 的最新 API 调整
	// 参考: https://github.com/go-pay/gopay/tree/main/alipay

	// 获取配置参数
	appID, _ := config["app_id"].(string)
	privateKey, _ := config["private_key"].(string)
	// appCertPublicKey, _ := config["app_cert_public_key"].(string)
	// alipayRootCert, _ := config["alipay_root_cert"].(string)
	// alipayPublicCert, _ := config["alipay_public_cert"].(string)

	if appID == "" || privateKey == "" {
		return nil, apperrors.ErrPaymentConfigRequired.WithMessage("支付宝配置不完整")
	}

	// 创建支付宝客户端
	client, err := alipay.NewClient(appID, privateKey, false)
	if err != nil {
		return nil, apperrors.ErrCreatePaymentFailed.WithError(err)
	}

	// 设置证书（如果需要）
	// 注意：这里需要根据 gopay 的最新 API 设置证书
	// 示例代码，实际使用时需要根据 gopay 的最新文档调整
	// if appCertPublicKey != "" {
	// 	client.SetAppCertPublicKey(appCertPublicKey)
	// }
	// if alipayRootCert != "" {
	// 	client.SetAlipayRootCert(alipayRootCert)
	// }
	// if alipayPublicCert != "" {
	// 	client.SetAlipayPublicCert(alipayPublicCert)
	// }

	// 设置回调地址
	notifyURL, _ := config["notify_url"].(string)
	if notifyURL == "" {
		notifyURL = fmt.Sprintf("%s/api/payment/notify/alipay", facades.Config().GetString("app.url"))
	}
	client.SetNotifyUrl(notifyURL)

	// 创建支付订单
	bm := make(gopay.BodyMap)
	bm.Set("out_trade_no", payment.PaymentNo)
	bm.Set("subject", payment.Remark)
	bm.Set("total_amount", fmt.Sprintf("%.2f", payment.Amount))
	bm.Set("product_code", "QUICK_MSECURITY_PAY")

	ctx := context.Background()
	payUrl, err := client.TradeAppPay(ctx, bm)
	if err != nil {
		return nil, apperrors.ErrCreatePaymentFailed.WithError(err)
	}

	return map[string]any{
		"payment_no": payment.PaymentNo,
		"pay_url":    payUrl,
	}, nil
}

// QueryPaymentOrder 查询支付订单状态
func (s *PaymentServiceImpl) QueryPaymentOrder(payment *models.Payment) (map[string]any, error) {
	// 获取支付方式
	paymentMethod, err := s.GetPaymentMethodByID(payment.PaymentMethodID)
	if err != nil {
		return nil, err
	}

	// 解析配置
	var config map[string]any
	if err := json.Unmarshal([]byte(paymentMethod.Config), &config); err != nil {
		return nil, apperrors.ErrPaymentConfigRequired.WithError(err)
	}

	// 根据支付类型查询
	switch paymentMethod.Type {
	case "wechat":
		return s.queryWechatPayment(payment, config)
	case "alipay":
		return s.queryAlipayPayment(payment, config)
	default:
		return nil, apperrors.ErrInvalidPaymentType.WithMessage(fmt.Sprintf("不支持的支付类型: %s", paymentMethod.Type))
	}
}

// queryWechatPayment 查询微信支付订单状态
func (s *PaymentServiceImpl) queryWechatPayment(payment *models.Payment, config map[string]any) (map[string]any, error) {
	// 实现微信支付查询逻辑
	// 这里需要根据 gopay 的微信支付文档实现
	return nil, fmt.Errorf("微信支付查询功能待实现")
}

// queryAlipayPayment 查询支付宝支付订单状态
func (s *PaymentServiceImpl) queryAlipayPayment(payment *models.Payment, config map[string]any) (map[string]any, error) {
	// 实现支付宝支付查询逻辑
	// 这里需要根据 gopay 的支付宝文档实现
	return nil, fmt.Errorf("支付宝支付查询功能待实现")
}

// HandlePaymentNotify 处理支付回调通知
func (s *PaymentServiceImpl) HandlePaymentNotify(paymentMethod *models.PaymentMethod, notifyData map[string]any) (*models.Payment, error) {
	// 根据支付类型处理回调
	switch paymentMethod.Type {
	case "wechat":
		return s.handleWechatNotify(paymentMethod, notifyData)
	case "alipay":
		return s.handleAlipayNotify(paymentMethod, notifyData)
	default:
		return nil, apperrors.ErrInvalidPaymentType.WithMessage(fmt.Sprintf("不支持的支付类型: %s", paymentMethod.Type))
	}
}

// handleWechatNotify 处理微信支付回调
func (s *PaymentServiceImpl) handleWechatNotify(paymentMethod *models.PaymentMethod, notifyData map[string]any) (*models.Payment, error) {
	// 实现微信支付回调处理逻辑
	// 这里需要根据 gopay 的微信支付文档实现
	return nil, fmt.Errorf("微信支付回调处理功能待实现")
}

// handleAlipayNotify 处理支付宝支付回调
func (s *PaymentServiceImpl) handleAlipayNotify(paymentMethod *models.PaymentMethod, notifyData map[string]any) (*models.Payment, error) {
	// 实现支付宝支付回调处理逻辑
	// 这里需要根据 gopay 的支付宝文档实现
	return nil, fmt.Errorf("支付宝支付回调处理功能待实现")
}

// buildPaymentMethodWhereClause 构建支付方式查询的 WHERE 条件
func (s *PaymentServiceImpl) buildPaymentMethodWhereClause(filters PaymentMethodFilters) (string, []any) {
	var conditions []string
	var args []any

	if filters.Name != "" {
		conditions = append(conditions, "name LIKE ?")
		args = append(args, "%"+filters.Name+"%")
	}
	if filters.Code != "" {
		conditions = append(conditions, "code = ?")
		args = append(args, filters.Code)
	}
	if filters.Type != "" {
		conditions = append(conditions, "type = ?")
		args = append(args, filters.Type)
	}
	if filters.IsActive != "" {
		switch filters.IsActive {
		case "1":
			conditions = append(conditions, "is_active = ?")
			args = append(args, true)
		case "0":
			conditions = append(conditions, "is_active = ?")
			args = append(args, false)
		}
	}
	if filters.Description != "" {
		conditions = append(conditions, "description LIKE ?")
		args = append(args, "%"+filters.Description+"%")
	}

	if len(conditions) == 0 {
		return "", nil
	}
	return strings.Join(conditions, " AND "), args
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
		if facades.Schema().HasTable(tableName) {
			var payment models.Payment
			if err := appfacades.OrmQuery(s.ctx).Model(&models.Payment{}).Table(tableName).Where("payment_no", paymentNo).First(&payment); err == nil {
				return &payment, nil
			}
		}
	}

	// 如果无法从支付单号解析日期，或者在对应分表中找不到，遍历最近6个月的分表
	now := time.Now().UTC()
	startTime := now.AddDate(0, -6, 0)
	tableNames := utils.GetShardingTableNames("payments", startTime, now)

	// 从最新的分表开始查询（Model 自动应用软删除过滤）
	for i := len(tableNames) - 1; i >= 0; i-- {
		if !facades.Schema().HasTable(tableNames[i]) {
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

	// 查询最近6个月的分表（足够覆盖大部分场景）
	now := time.Now().UTC()
	startTime := now.AddDate(0, -6, 0)
	tableNames := utils.GetShardingTableNames("payments", startTime, now)

	// 从最新的分表开始查询（Model 自动应用软删除过滤）
	for i := len(tableNames) - 1; i >= 0; i-- {
		if !facades.Schema().HasTable(tableNames[i]) {
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
	if !facades.Schema().HasTable(tableName) {
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

	// 检查分表是否存在
	if !facades.Schema().HasTable(tableName) {
		// 使用迁移函数创建分表
		if err := migrations.CreatePaymentsShardingTable(tableName); err != nil {
			errorlog.Record(s.ctx, "payment", "创建支付记录分表失败", map[string]any{
				"table_name": tableName,
				"error":      err.Error(),
			}, "创建支付记录分表失败: %v", err)
			return "", fmt.Errorf("创建支付记录分表失败: %v", err)
		}
	}

	return tableName, nil
}
