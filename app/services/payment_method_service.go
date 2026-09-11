package services

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/dromara/carbon/v2"
	"github.com/goravel/framework/contracts/database/orm"
	"github.com/goravel/framework/contracts/http"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	admin "goravel/app/http/requests/admin"
	"goravel/app/models"
	"goravel/app/utils"
)

// PaymentMethodService 支付方式 CRUD（与支付记录/网关解耦）。
type PaymentMethodService interface {
	GetPaymentMethodByID(id uint) (*models.PaymentMethod, error)
	GetPaymentMethodByCode(code string) (*models.PaymentMethod, error)
	GetPaymentMethods(filters PaymentMethodFilters, page, pageSize int) ([]models.PaymentMethod, int64, error)
	CreatePaymentMethod(name, code, paymentType string, config map[string]any, isActive bool, sort int, description string) (*models.PaymentMethod, error)
	CreatePaymentMethodFromRequest(req *admin.PaymentMethodCreate) (*models.PaymentMethod, error)
	UpdatePaymentMethod(id uint, name string, config map[string]any, isActive bool, sort int, description string) error
	UpdatePaymentMethodModel(paymentMethod *models.PaymentMethod) error
	UpdatePaymentMethodByRequest(httpCtx http.Context, id uint, req *admin.PaymentMethodUpdate) (*models.PaymentMethod, error)
	DeletePaymentMethod(id uint) error
	PaymentMethodListItem(pm models.PaymentMethod) map[string]any
	PaymentMethodDetail(pm *models.PaymentMethod) map[string]any
}

type PaymentMethodServiceImpl struct {
	ctx context.Context
}

func NewPaymentMethodService(ctx context.Context) PaymentMethodService {
	return &PaymentMethodServiceImpl{ctx: ctx}
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

func (s *PaymentMethodServiceImpl) GetPaymentMethodByID(id uint) (*models.PaymentMethod, error) {
	var paymentMethod models.PaymentMethod
	if err := appfacades.OrmQuery(s.ctx).Where("id", id).FirstOrFail(&paymentMethod); err != nil {
		return nil, apperrors.ErrPaymentMethodNotFound.WithError(err)
	}
	return &paymentMethod, nil
}

// GetPaymentMethodByCode 根据代码获取支付方式
func (s *PaymentMethodServiceImpl) GetPaymentMethodByCode(code string) (*models.PaymentMethod, error) {
	var paymentMethod models.PaymentMethod
	if err := appfacades.OrmQuery(s.ctx).Where("code", code).Where("is_active", true).FirstOrFail(&paymentMethod); err != nil {
		return nil, apperrors.ErrPaymentMethodNotFound.WithError(err)
	}
	return &paymentMethod, nil
}

// GetPaymentMethods 获取支付方式列表
func (s *PaymentMethodServiceImpl) GetPaymentMethods(filters PaymentMethodFilters, page, pageSize int) ([]models.PaymentMethod, int64, error) {
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
	if filters.Description != "" {
		query = query.Where("description LIKE ?", "%"+filters.Description+"%")
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
func (s *PaymentMethodServiceImpl) validatePaymentMethodCodeUnique(code string, excludeID uint) error {
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
func (s *PaymentMethodServiceImpl) CreatePaymentMethod(name, code, paymentType string, config map[string]any, isActive bool, sort int, description string) (*models.PaymentMethod, error) {
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
func (s *PaymentMethodServiceImpl) CreatePaymentMethodFromRequest(req *admin.PaymentMethodCreate) (*models.PaymentMethod, error) {
	return s.CreatePaymentMethod(req.Name, req.Code, req.Type, req.Config, req.IsActive, req.Sort, req.Description)
}

// UpdatePaymentMethod 更新支付方式
func (s *PaymentMethodServiceImpl) UpdatePaymentMethod(id uint, name string, config map[string]any, isActive bool, sort int, description string) error {
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
func (s *PaymentMethodServiceImpl) UpdatePaymentMethodModel(paymentMethod *models.PaymentMethod) error {
	if err := appfacades.OrmQuery(s.ctx).Save(paymentMethod); err != nil {
		return apperrors.ErrUpdateFailed.WithError(err)
	}
	return nil
}

// UpdatePaymentMethodByRequest 按请求部分更新支付方式
func (s *PaymentMethodServiceImpl) UpdatePaymentMethodByRequest(httpCtx http.Context, id uint, req *admin.PaymentMethodUpdate) (*models.PaymentMethod, error) {
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
func (s *PaymentMethodServiceImpl) DeletePaymentMethod(id uint) error {
	_, err := s.GetPaymentMethodByID(id)
	if err != nil {
		return err
	}

	if _, err := appfacades.OrmQuery(s.ctx).Where("id", id).Delete(&models.PaymentMethod{}); err != nil {
		return apperrors.ErrDeleteFailed.WithError(err)
	}

	return nil
}

func (s *PaymentMethodServiceImpl) PaymentMethodListItem(pm models.PaymentMethod) map[string]any {
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

func (s *PaymentMethodServiceImpl) PaymentMethodDetail(pm *models.PaymentMethod) map[string]any {
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

func (s *PaymentMethodServiceImpl) applyOrderBy(query orm.Query, orderBy string) orm.Query {
	if orderBy == "" {
		return query
	}
	parts := strings.Split(orderBy, ":")
	if len(parts) != 2 {
		return query
	}
	field, direction := parts[0], strings.ToLower(parts[1])
	allowed := map[string]bool{
		"id": true, "name": true, "code": true, "type": true,
		"sort": true, "created_at": true, "updated_at": true,
	}
	if !allowed[field] {
		return query
	}
	if direction != "asc" && direction != "desc" {
		direction = "desc"
	}
	return query.Order(field + " " + direction)
}
