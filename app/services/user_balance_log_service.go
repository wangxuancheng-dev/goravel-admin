package services

import (
	"context"
	appfacades "goravel/app/facades"
	"time"

	"github.com/goravel/framework/contracts/database/orm"

	apperrors "goravel/app/errors"
	"goravel/app/models"
	"goravel/app/utils"
	"goravel/app/utils/errorlog"
)

type UserBalanceLogService interface {
	// CreateLog 创建余额变动记录（使用自定义分表逻辑，必须提供 user_id）
	CreateLog(userID uint, logType string, amount float64, balance float64, source string, sourceID *uint, description string, operatorID *uint, status string, remark string) (*models.UserBalanceLog, error)
	// GetLogs 查询余额变动记录列表（必须提供 user_id）
	GetLogs(filters UserBalanceLogFilters, page, pageSize int) ([]models.UserBalanceLog, int64, error)
	// GetUserBalance 获取用户当前余额（从 users 表获取）
	GetUserBalance(userID uint) (float64, error)
	// GetUserStatistics 获取用户余额统计
	GetUserStatistics(userID uint, startTime, endTime time.Time) (*UserBalanceStatistics, error)
}

type UserBalanceLogFilters struct {
	UserID     uint      // 用户ID（必填，用于分表路由）
	Type       string    // 变动类型
	Source     string    // 来源
	Status     string    // 状态
	StartTime  time.Time // 开始时间
	EndTime    time.Time // 结束时间
	OperatorID *uint     // 操作员ID
}

type UserBalanceStatistics struct {
	TotalIncome    float64 `json:"total_income"`    // 总收入
	TotalExpense   float64 `json:"total_expense"`   // 总支出
	TotalRefund    float64 `json:"total_refund"`    // 总退款
	CurrentBalance float64 `json:"current_balance"` // 当前余额
}

type UserBalanceLogServiceImpl struct {
	ctx             context.Context
	shardingService ShardingService
}

func NewUserBalanceLogService(ctx context.Context) UserBalanceLogService {
	return &UserBalanceLogServiceImpl{
		ctx:             ctx,
		shardingService: NewShardingService(ctx),
	}
}

// CreateLog 创建余额变动记录
// 注意：必须提供 user_id，会根据 user_id 自动路由到对应分表
func (s *UserBalanceLogServiceImpl) CreateLog(
	userID uint,
	logType string,
	amount float64,
	balance float64,
	source string,
	sourceID *uint,
	description string,
	operatorID *uint,
	status string,
	remark string,
) (*models.UserBalanceLog, error) {
	if userID == 0 {
		return nil, apperrors.ErrUserIDRequired.WithMessage("user_id 不能为空")
	}

	// 默认状态为 success
	if status == "" {
		status = "success"
	}

	// 根据 user_id 计算分表名称
	tableName := utils.GetHashShardingTableNameByConfig(utils.GetUserBalanceLogsShardingConfig(), userID)

	// 检查分表是否存在，不存在则创建
	if err := s.shardingService.EnsureShardingTable(tableName, "user_balance_logs"); err != nil {
		errorlog.Record(s.ctx, "user-balance-log", "创建分表失败", map[string]any{
			"table_name": tableName,
			"user_id":    userID,
			"error":      err.Error(),
		}, "创建分表 %s 失败: %v", tableName, err)
		return nil, apperrors.ErrCreateFailed.WithError(err)
	}

	log := &models.UserBalanceLog{
		UserID:      userID,
		Type:        logType,
		Amount:      amount,
		Balance:     balance,
		Source:      source,
		SourceID:    sourceID,
		Description: description,
		OperatorID:  operatorID,
		Status:      status,
		Remark:      remark,
	}

	// 使用 Goravel ORM，通过 Table() 方法指定分表名称
	err := appfacades.OrmQuery(s.ctx).Table(tableName).Create(log)
	if err != nil {
		errorlog.Record(s.ctx, "user-balance-log", "创建余额变动记录失败", map[string]any{
			"user_id":    userID,
			"log_type":   logType,
			"amount":     amount,
			"table_name": tableName,
			"error":      err.Error(),
		}, "创建余额变动记录失败: %v", err)
		return nil, apperrors.ErrCreateFailed.WithError(err)
	}

	return log, nil
}

// GetLogs 查询余额变动记录列表
// 注意：必须提供 user_id，会根据 user_id 自动路由到对应分表
func (s *UserBalanceLogServiceImpl) GetLogs(filters UserBalanceLogFilters, page, pageSize int) ([]models.UserBalanceLog, int64, error) {
	if filters.UserID == 0 {
		return nil, 0, apperrors.ErrUserIDRequired.WithMessage("user_id 不能为空")
	}

	// 根据 user_id 计算分表名称
	tableName := utils.GetHashShardingTableNameByConfig(utils.GetUserBalanceLogsShardingConfig(), filters.UserID)

	// 构建基础查询（用于 Count 和 Get）
	buildQuery := func() orm.Query {
		query := appfacades.OrmQuery(s.ctx).Table(tableName).
			Where("user_id", filters.UserID)

		// 添加其他筛选条件
		if filters.Type != "" {
			query = query.Where("type", filters.Type)
		}
		if filters.Source != "" {
			query = query.Where("source", filters.Source)
		}
		if filters.Status != "" {
			query = query.Where("status", filters.Status)
		}
		if !filters.StartTime.IsZero() {
			query = query.Where("created_at >= ?", filters.StartTime)
		}
		if !filters.EndTime.IsZero() {
			query = query.Where("created_at <= ?", filters.EndTime)
		}
		if filters.OperatorID != nil && *filters.OperatorID > 0 {
			query = query.Where("operator_id", *filters.OperatorID)
		}

		return query
	}

	// 获取总数
	total, err := buildQuery().Count()
	if err != nil {
		return nil, 0, apperrors.ErrQueryFailed.WithError(err)
	}

	// 分页查询
	// 使用 Find() 而不是 Get()，因为 Find() 会保持 Table() 设置的表名
	var logs []models.UserBalanceLog
	err = buildQuery().
		Order("created_at desc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&logs)
	if err != nil {
		return nil, 0, apperrors.ErrQueryFailed.WithError(err)
	}

	return logs, total, nil
}

// GetUserBalance 获取用户当前余额（从 users 表获取）
func (s *UserBalanceLogServiceImpl) GetUserBalance(userID uint) (float64, error) {
	if userID == 0 {
		return 0, apperrors.ErrUserIDRequired
	}

	var user models.User
	err := appfacades.OrmQuery(s.ctx).Where("id", userID).First(&user)
	if err != nil {
		return 0, apperrors.ErrUserNotFound.WithError(err)
	}

	return user.Balance, nil
}

// GetUserStatistics 获取用户余额统计
func (s *UserBalanceLogServiceImpl) GetUserStatistics(userID uint, startTime, endTime time.Time) (*UserBalanceStatistics, error) {
	if userID == 0 {
		return nil, apperrors.ErrUserIDRequired
	}

	// 根据 user_id 计算分表名称
	tableName := utils.GetHashShardingTableNameByConfig(utils.GetUserBalanceLogsShardingConfig(), userID)

	var stats UserBalanceStatistics

	// 构建基础查询条件（用于构建 SQL）
	baseConditions := []any{userID, "success"}
	baseWhere := "user_id = ? AND status = ?"
	if !startTime.IsZero() {
		baseWhere += " AND created_at >= ?"
		baseConditions = append(baseConditions, startTime)
	}
	if !endTime.IsZero() {
		baseWhere += " AND created_at <= ?"
		baseConditions = append(baseConditions, endTime)
	}

	// 统计收入
	var incomeResult struct {
		Total float64
	}
	incomeSQL := "SELECT COALESCE(SUM(amount), 0) as total FROM " + tableName + " WHERE " + baseWhere + " AND type = ?"
	incomeArgs := append(baseConditions, "income")
	if err := appfacades.OrmQuery(s.ctx).Raw(incomeSQL, incomeArgs...).Scan(&incomeResult); err == nil {
		stats.TotalIncome = incomeResult.Total
	}

	// 统计支出
	var expenseResult struct {
		Total float64
	}
	expenseSQL := "SELECT COALESCE(SUM(amount), 0) as total FROM " + tableName + " WHERE " + baseWhere + " AND type = ?"
	expenseArgs := append(baseConditions, "expense")
	if err := appfacades.OrmQuery(s.ctx).Raw(expenseSQL, expenseArgs...).Scan(&expenseResult); err == nil {
		stats.TotalExpense = expenseResult.Total
	}

	// 统计退款
	var refundResult struct {
		Total float64
	}
	refundSQL := "SELECT COALESCE(SUM(amount), 0) as total FROM " + tableName + " WHERE " + baseWhere + " AND type = ?"
	refundArgs := append(baseConditions, "refund")
	if err := appfacades.OrmQuery(s.ctx).Raw(refundSQL, refundArgs...).Scan(&refundResult); err == nil {
		stats.TotalRefund = refundResult.Total
	}

	// 获取当前余额
	balance, err := s.GetUserBalance(userID)
	if err == nil {
		// 获取用户信息以获取货币信息
		var user models.User
		if err := appfacades.OrmQuery(s.ctx).Where("id", userID).First(&user); err == nil {
			// 加载货币信息
			if user.CurrencyID > 0 {
				var currency models.Currency
				if err := appfacades.OrmQuery(s.ctx).Where("id", user.CurrencyID).First(&currency); err == nil {
					user.Currency = &currency
				}
			}
			stats.CurrentBalance = utils.FormatBalance(balance, user.Currency)
			// 格式化统计金额
			stats.TotalIncome = utils.FormatBalance(stats.TotalIncome, user.Currency)
			stats.TotalExpense = utils.FormatBalance(stats.TotalExpense, user.Currency)
			stats.TotalRefund = utils.FormatBalance(stats.TotalRefund, user.Currency)
		} else {
			stats.CurrentBalance = balance
		}
	}

	return &stats, nil
}
