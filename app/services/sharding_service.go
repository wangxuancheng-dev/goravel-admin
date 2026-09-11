package services

import (
	"context"

	"github.com/goravel/framework/facades"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/utils"
	"goravel/app/utils/errorlog"
	"goravel/database/migrations"
)

// TableCreator 表创建函数类型
type TableCreator func(tableName string) error

// ShardingService 分表服务接口
type ShardingService interface {
	// RegisterTableCreator 注册表的创建函数
	RegisterTableCreator(baseTableName string, creator TableCreator)
	// CreateShardingTable 创建分表
	CreateShardingTable(tableName, baseTableName string) error
	// EnsureShardingTable 确保分表存在，不存在则创建
	EnsureShardingTable(tableName, baseTableName string) error
}

type ShardingServiceImpl struct {
	ctx      context.Context
	creators map[string]TableCreator
}

func NewShardingService(ctx context.Context) ShardingService {
	service := &ShardingServiceImpl{
		ctx:      ctx,
		creators: make(map[string]TableCreator),
	}

	service.registerOrderTables()
	service.registerPaymentTables()
	service.registerUserBalanceLogTables()

	return service
}

func (s *ShardingServiceImpl) registerOrderTables() {
	s.RegisterTableCreator("orders", migrations.CreateOrdersShardingTable)
	s.RegisterTableCreator("order_details", migrations.CreateOrderDetailsShardingTable)
}

func (s *ShardingServiceImpl) registerPaymentTables() {
	s.RegisterTableCreator("payments", migrations.CreatePaymentsShardingTable)
}

func (s *ShardingServiceImpl) registerUserBalanceLogTables() {
	s.RegisterTableCreator("user_balance_logs", migrations.CreateUserBalanceLogsShardingTable)
}

func (s *ShardingServiceImpl) RegisterTableCreator(baseTableName string, creator TableCreator) {
	s.creators[baseTableName] = creator
}

func (s *ShardingServiceImpl) CreateShardingTable(tableName, baseTableName string) error {
	creator, exists := s.creators[baseTableName]
	if !exists {
		return apperrors.ErrBaseTableNotRegistered.WithParams(map[string]any{
			"base_table_name": baseTableName,
		})
	}
	// DDL 走 Schema；HTTP 路径需按 ctx 切到租户连接
	return appfacades.WithSchemaContext(s.ctx, func() error {
		return creator(tableName)
	})
}

func (s *ShardingServiceImpl) EnsureShardingTable(tableName, baseTableName string) error {
	if utils.ShardingTableExistsCtx(s.ctx, tableName) {
		return nil
	}

	return utils.WithShardingTableCreateLockCtx(s.ctx, tableName, func() error {
		if utils.ShardingTableExistsCtx(s.ctx, tableName) {
			return nil
		}

		if err := s.CreateShardingTable(tableName, baseTableName); err != nil {
			if utils.ShardingTableExistsCtx(s.ctx, tableName) || appfacades.SchemaHasTable(s.ctx, tableName) {
				utils.MarkShardingTableExistsCtx(s.ctx, tableName)
				return nil
			}
			errorlog.Record(s.ctx, "sharding", "创建分表失败", map[string]any{
				"table_name":      tableName,
				"base_table_name": baseTableName,
				"error":           err.Error(),
			}, "创建分表 %s 失败: %v", tableName, err)
			return apperrors.ErrCreateShardingTableFailed.WithError(err).WithParams(map[string]any{
				"table_name": tableName,
			})
		}

		utils.MarkShardingTableExistsCtx(s.ctx, tableName)
		facades.Log().Infof("自动创建分表: %s", tableName)
		return nil
	})
}
