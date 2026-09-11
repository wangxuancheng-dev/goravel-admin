package services

import (
	"context"
	"fmt"
	appfacades "goravel/app/facades"
	"time"

	"github.com/dromara/carbon/v2"
	"github.com/goravel/framework/contracts/database/orm"
	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"

	apperrors "goravel/app/errors"
	"goravel/app/http/requests/admin"
	"goravel/app/models"
	"goravel/app/tenancy"
	"goravel/app/utils"
	"goravel/app/utils/errorlog"
)

type UserService interface {
	GetByID(id uint) (*models.User, error)
	GetList(filters UserFilters, page, pageSize int) ([]models.User, int64, error)
	Create(req *admin.UserCreate) (*models.User, error)
	Update(id uint, req *admin.UserUpdate) (*models.User, error)
	Delete(id uint) error
	UpdateBalance(userID uint, amount float64, logType string, source string, sourceID *uint, description string, operatorID *uint, remark string) error
	ResetPassword(userID uint, newPassword string) error
}

type UserFilters struct {
	Username string
	Email    string
	Phone    string
	Nickname string
	Status   string
}

// BuildUserFiltersFromHTTP reads list filters from query or body.
func BuildUserFiltersFromHTTP(ctx http.Context) UserFilters {
	return UserFilters{
		Username: ctx.Request().Input("username", ctx.Request().Query("username", "")),
		Nickname: ctx.Request().Input("nickname", ctx.Request().Query("nickname", "")),
		Email:    ctx.Request().Input("email", ctx.Request().Query("email", "")),
		Phone:    ctx.Request().Input("phone", ctx.Request().Query("phone", "")),
		Status:   ctx.Request().Input("status", ctx.Request().Query("status", "")),
	}
}

// BuildUserQuery 构建用户查询（通用查询构建，供列表和导出复用）
func BuildUserQuery(ctx context.Context, filters UserFilters) orm.Query {
	query := appfacades.OrmQuery(ctx).Model(&models.User{})

	if filters.Username != "" {
		query = query.Where("username LIKE ?", "%"+filters.Username+"%")
	}
	if filters.Nickname != "" {
		query = query.Where("nickname LIKE ?", "%"+filters.Nickname+"%")
	}
	if filters.Email != "" {
		query = query.Where("email LIKE ?", "%"+filters.Email+"%")
	}
	if filters.Phone != "" {
		query = query.Where("phone LIKE ?", "%"+filters.Phone+"%")
	}
	if filters.Status != "" {
		query = query.Where("status", filters.Status)
	}

	return query
}

type UserServiceImpl struct {
	ctx               context.Context
	balanceLogService UserBalanceLogService
}

func NewUserService(ctx context.Context) UserService {
	return &UserServiceImpl{
		ctx:               ctx,
		balanceLogService: NewUserBalanceLogService(ctx),
	}
}

// GetByID 根据ID获取用户
func (s *UserServiceImpl) GetByID(id uint) (*models.User, error) {
	var user models.User
	if err := appfacades.OrmQuery(s.ctx).Where("id", id).FirstOrFail(&user); err != nil {
		return nil, apperrors.ErrUserNotFound.WithError(err)
	}

	// 加载货币信息
	if user.CurrencyID > 0 {
		var currency models.Currency
		if err := appfacades.OrmQuery(s.ctx).Where("id", user.CurrencyID).First(&currency); err == nil {
			user.Currency = &currency
		}
	}

	// 格式化余额
	user.Balance = utils.FormatBalance(user.Balance, user.Currency)
	return &user, nil
}

// GetList 获取用户列表
func (s *UserServiceImpl) GetList(filters UserFilters, page, pageSize int) ([]models.User, int64, error) {
	query := BuildUserQuery(s.ctx, filters)

	// 分页查询
	var users []models.User
	var total int64
	if err := query.Order("created_at desc").Paginate(page, pageSize, &users, &total); err != nil {
		return nil, 0, err
	}

	// 批量加载货币信息，避免 N+1 查询
	currencyMap := make(map[uint]*models.Currency)
	currencyIDs := make([]uint, 0, len(users))
	seenCurrencyIDs := make(map[uint]struct{})
	for i := range users {
		if users[i].CurrencyID == 0 {
			continue
		}
		if _, exists := seenCurrencyIDs[users[i].CurrencyID]; exists {
			continue
		}
		seenCurrencyIDs[users[i].CurrencyID] = struct{}{}
		currencyIDs = append(currencyIDs, users[i].CurrencyID)
	}
	if len(currencyIDs) > 0 {
		var currencies []models.Currency
		if err := appfacades.OrmQuery(s.ctx).Where("id IN ?", currencyIDs).Find(&currencies); err == nil {
			for i := range currencies {
				c := currencies[i]
				currencyMap[c.ID] = &c
			}
		}
	}
	for i := range users {
		if currency, exists := currencyMap[users[i].CurrencyID]; exists {
			users[i].Currency = currency
		}
		users[i].Balance = utils.FormatBalance(users[i].Balance, users[i].Currency)
	}

	return users, total, nil
}

// createRecord persists a user row (internal helper).
func (s *UserServiceImpl) createRecord(user *models.User) error {
	// 如果未设置货币ID，默认使用人民币
	if user.CurrencyID == 0 {
		var cnyCurrency models.Currency
		if err := appfacades.OrmQuery(s.ctx).Where("code", "CNY").First(&cnyCurrency); err == nil {
			user.CurrencyID = cnyCurrency.ID
		}
	}
	// 使用 map 创建，确保 Status 为 0 时也能正确保存（与管理员创建方式一致）
	// GORM 在处理结构体时可能会忽略零值字段，使用 map 可以确保所有字段都被保存
	userData := map[string]any{
		"username":      user.Username,
		"password":      user.Password,
		"nickname":      user.Nickname,
		"avatar":        user.Avatar,
		"email":         user.Email,
		"phone":         user.Phone,
		"balance":       user.Balance,
		"currency_id":   user.CurrencyID,
		"status":        user.Status,
		"last_login_at": user.LastLoginAt,
		"created_at":    carbon.Now(),
		"updated_at":    carbon.Now(),
	}
	if err := appfacades.OrmQuery(s.ctx).Model(&models.User{}).Create(userData); err != nil {
		return err
	}
	// 将创建后的 ID 赋值回 user 对象（GORM 会将生成的 ID 填充到 map 中）
	if id, ok := userData["id"].(uint); ok {
		user.ID = id
	} else if id, ok := userData["id"].(uint64); ok {
		user.ID = uint(id)
	} else {
		// 如果 map 中没有 ID，通过用户名查询获取（与管理员创建方式一致）
		var createdUser models.User
		if err := appfacades.OrmQuery(s.ctx).Where("username", user.Username).First(&createdUser); err == nil {
			user.ID = createdUser.ID
		}
	}
	return nil
}

func (s *UserServiceImpl) Create(req *admin.UserCreate) (*models.User, error) {
	if err := s.validateUserExists(req.Username, req.Email, req.Phone, 0); err != nil {
		return nil, err
	}

	hashedPassword, err := facades.Hash().Make(req.Password)
	if err != nil {
		return nil, apperrors.NewBusinessError("password_encrypt_failed", "密码加密失败").WithError(err)
	}

	var currencyID uint
	var cnyCurrency models.Currency
	if err := appfacades.OrmQuery(s.ctx).Where("code", "CNY").First(&cnyCurrency); err == nil {
		currencyID = cnyCurrency.ID
	}

	user := &models.User{
		Username:   req.Username,
		Password:   hashedPassword,
		Nickname:   req.Nickname,
		Email:      req.Email,
		Phone:      req.Phone,
		Balance:    0,
		CurrencyID: currencyID,
		Status:     req.Status,
	}

	if err := s.createRecord(user); err != nil {
		return nil, apperrors.ErrCreateFailed.WithError(err)
	}

	return s.GetByID(user.ID)
}

func (s *UserServiceImpl) Update(id uint, req *admin.UserUpdate) (*models.User, error) {
	user, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	email := user.Email
	if req.Email != nil {
		email = *req.Email
	}
	phone := user.Phone
	if req.Phone != nil {
		phone = *req.Phone
	}
	if err := s.validateUserExists("", email, phone, id); err != nil {
		return nil, err
	}

	updateData := map[string]any{}
	if req.Nickname != nil {
		updateData["nickname"] = *req.Nickname
	}
	if req.Email != nil {
		updateData["email"] = *req.Email
	}
	if req.Phone != nil {
		updateData["phone"] = *req.Phone
	}
	if req.Status != nil {
		updateData["status"] = *req.Status
	}
	if req.Password != nil && *req.Password != "" {
		hashedPassword, err := facades.Hash().Make(*req.Password)
		if err != nil {
			return nil, apperrors.NewBusinessError("password_encrypt_failed", "密码加密失败").WithError(err)
		}
		updateData["password"] = hashedPassword
	}

	if len(updateData) == 0 {
		return user, nil
	}

	if _, err := appfacades.OrmQuery(s.ctx).Model(&models.User{}).Where("id", id).Update(updateData); err != nil {
		return nil, apperrors.ErrUpdateFailed.WithError(err)
	}

	return s.GetByID(id)
}

// Delete 删除用户（软删除）
func (s *UserServiceImpl) Delete(id uint) error {
	if _, err := s.GetByID(id); err != nil {
		return err
	}
	if _, err := appfacades.OrmQuery(s.ctx).Where("id", id).Delete(&models.User{}); err != nil {
		return apperrors.ErrDeleteFailed.WithError(err)
	}
	return nil
}

// UpdateBalance 更新用户余额（同时创建余额变动记录）
func (s *UserServiceImpl) UpdateBalance(userID uint, amount float64, logType string, source string, sourceID *uint, description string, operatorID *uint, remark string) error {
	lockKey := tenancy.CacheKey(s.ctx, fmt.Sprintf("user:balance:lock:%d", userID))
	lock := facades.Cache().Lock(lockKey, 10*time.Second)
	if !lock.Get() {
		return apperrors.NewBusinessError("too_many_requests", "请求过于频繁，请稍后再试")
	}
	defer lock.Release()

	// 获取当前用户
	user, err := s.GetByID(userID)
	if err != nil {
		return err
	}

	// 计算新余额
	var newBalance float64
	switch logType {
	case "income", "refund":
		newBalance = user.Balance + amount
	case "expense":
		newBalance = user.Balance - amount
		if newBalance < 0 {
			return apperrors.ErrInsufficientBalance.WithParams(map[string]any{
				"balance": user.Balance,
			})
		}
	default:
		return apperrors.ErrInvalidBalanceType.WithParams(map[string]any{
			"type": logType,
		})
	}

	// 1. 更新用户余额
	_, err = appfacades.OrmQuery(s.ctx).Model(&models.User{}).Where("id", userID).Update(map[string]any{
		"balance": newBalance,
	})
	if err != nil {
		errorlog.Record(s.ctx, "user", "更新用户余额失败", map[string]any{
			"user_id":     userID,
			"amount":      amount,
			"log_type":    logType,
			"new_balance": newBalance,
			"error":       err.Error(),
		}, "更新用户余额失败: %v", err)
		return apperrors.ErrUpdateFailed.WithError(err)
	}

	// 2. 创建余额变动记录（使用自定义分表逻辑）
	_, err = s.balanceLogService.CreateLog(userID, logType, amount, newBalance, source, sourceID, description, operatorID, "success", remark)
	if err != nil {
		// 如果创建记录失败，尝试回滚余额更新
		// 注意：由于涉及分表，无法使用跨表事务，这里手动回滚
		rollbackErr := s.rollbackBalance(userID, user.Balance)
		if rollbackErr != nil {
			errorlog.Record(s.ctx, "user", "回滚余额失败", map[string]any{
				"user_id":      userID,
				"old_balance":  user.Balance,
				"new_balance":  newBalance,
				"rollback_err": rollbackErr.Error(),
			}, "回滚余额失败: %v", rollbackErr)
		}

		errorlog.Record(s.ctx, "user", "创建余额变动记录失败", map[string]any{
			"user_id":     userID,
			"amount":      amount,
			"log_type":    logType,
			"new_balance": newBalance,
			"error":       err.Error(),
		}, "创建余额变动记录失败: %v", err)
		return apperrors.ErrCreateFailed.WithError(err)
	}

	return nil
}

// rollbackBalance 回滚余额到指定值
func (s *UserServiceImpl) rollbackBalance(userID uint, balance float64) error {
	_, err := appfacades.OrmQuery(s.ctx).Model(&models.User{}).Where("id", userID).Update(map[string]any{
		"balance": balance,
	})
	return err
}

// validateUserExists 校验用户名/邮箱/手机号是否与活跃用户冲突（用户：软删后可复用）。
func (s *UserServiceImpl) validateUserExists(username, email, phone string, excludeID uint) error {
	checks := []struct {
		column string
		value  string
		err    error
	}{
		{"username", username, apperrors.ErrUsernameExists},
		{"email", email, apperrors.NewBusinessError("email_already_exists", "邮箱已存在")},
		{"phone", phone, apperrors.NewBusinessError("phone_already_exists", "手机号已存在")},
	}

	for _, check := range checks {
		if check.value == "" {
			continue
		}
		exists, err := utils.ExistsColumnValue(s.ctx, "users", &models.User{}, utils.UniqueReuseAllow, check.column, check.value, excludeID)
		if err != nil {
			return apperrors.ErrCreateFailed.WithError(err)
		}
		if exists {
			return check.err
		}
	}

	return nil
}

// ResetPassword 重置用户密码
func (s *UserServiceImpl) ResetPassword(userID uint, newPassword string) error {
	// 检查用户是否存在
	_, err := s.GetByID(userID)
	if err != nil {
		return err
	}

	// 密码加密
	hashedPassword, err := facades.Hash().Make(newPassword)
	if err != nil {
		return apperrors.ErrPasswordEncryptFailed.WithError(err)
	}

	// 更新密码
	_, err = appfacades.OrmQuery(s.ctx).Model(&models.User{}).Where("id", userID).Update(map[string]any{
		"password": hashedPassword,
	})
	if err != nil {
		return apperrors.ErrUpdateFailed.WithError(err)
	}

	return nil
}
