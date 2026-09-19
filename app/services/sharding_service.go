package services

import (
	"context"
	"fmt"
	"strings"

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
	// DDL via Schema; HTTP path must bind tenant connection from ctx.
	err := appfacades.WithSchemaContext(s.ctx, func() error {
		return creator(tableName)
	})
	if err != nil {
		// Concurrent create is OK only if the table exists on THIS tenant connection.
		// Error 1050 on the platform DB must not be treated as success.
		if isShardingTableAlreadyExistsError(err) && appfacades.SchemaHasTable(s.ctx, tableName) {
			return nil
		}
		return err
	}
	if !appfacades.SchemaHasTable(s.ctx, tableName) {
		return fmt.Errorf("sharding table %s create reported ok but missing on tenant connection", tableName)
	}
	return nil
}

func (s *ShardingServiceImpl) EnsureShardingTable(tableName, baseTableName string) error {
	// Prefer live Schema check: negative cache must not block create fallback.
	if appfacades.SchemaHasTable(s.ctx, tableName) {
		utils.MarkShardingTableExistsCtx(s.ctx, tableName)
		return nil
	}
	if utils.ShardingTableExistsCtx(s.ctx, tableName) {
		return nil
	}

	return utils.WithShardingTableCreateLockCtx(s.ctx, tableName, func() error {
		if appfacades.SchemaHasTable(s.ctx, tableName) {
			utils.MarkShardingTableExistsCtx(s.ctx, tableName)
			return nil
		}

		if err := s.CreateShardingTable(tableName, baseTableName); err != nil {
			utils.InvalidateShardingTableCacheCtx(s.ctx, tableName)
			if appfacades.SchemaHasTable(s.ctx, tableName) {
				utils.MarkShardingTableExistsCtx(s.ctx, tableName)
				return nil
			}
			errorlog.Record(s.ctx, "sharding", "create sharding table failed", map[string]any{
				"table_name":      tableName,
				"base_table_name": baseTableName,
				"error":           err.Error(),
			}, "create sharding table %s failed: %v", tableName, err)
			return apperrors.ErrCreateShardingTableFailed.WithError(err).WithParams(map[string]any{
				"table_name": tableName,
			})
		}

		utils.MarkShardingTableExistsCtx(s.ctx, tableName)
		facades.Log().Infof("auto-created sharding table: %s", tableName)
		return nil
	})
}

// isShardingTableAlreadyExistsError detects concurrent/prebuilt CREATE races (MySQL 1050, etc.).
func isShardingTableAlreadyExistsError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "already exists") ||
		strings.Contains(msg, "1050") ||
		strings.Contains(msg, "42s01") ||
		strings.Contains(msg, "duplicate table")
}
