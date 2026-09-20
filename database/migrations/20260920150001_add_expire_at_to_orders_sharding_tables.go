package migrations

import (
	"context"
	"fmt"

	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"

	"goravel/app/utils"
)

// M20260920150001AddExpireAtToOrdersShardingTables adds expire_at for unpaid order auto-cancel demo.
type M20260920150001AddExpireAtToOrdersShardingTables struct{}

func (r *M20260920150001AddExpireAtToOrdersShardingTables) Signature() string {
	return "20260920150001_add_expire_at_to_orders_sharding_tables"
}

func (r *M20260920150001AddExpireAtToOrdersShardingTables) Up() error {
	ordersTables, err := utils.GetAllExistingShardingTables(context.Background(), "orders")
	if err != nil {
		return fmt.Errorf("list order shards: %w", err)
	}
	if len(ordersTables) == 0 {
		facades.Log().Info("no order shards to alter for expire_at")
		return nil
	}

	var failed []string
	for _, tableName := range ordersTables {
		if !facades.Schema().HasTable(tableName) {
			continue
		}
		if facades.Schema().HasColumn(tableName, "expire_at") {
			continue
		}
		if err := facades.Schema().Table(tableName, func(table schema.Blueprint) {
			table.Timestamp("expire_at").Nullable().Comment("pending order auto-cancel deadline (UTC)")
			table.Index("status", "expire_at")
		}); err != nil {
			facades.Log().Errorf("add expire_at to %s failed: %v", tableName, err)
			failed = append(failed, tableName)
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("partial expire_at migration failed: %v", failed)
	}
	return nil
}

func (r *M20260920150001AddExpireAtToOrdersShardingTables) Down() error {
	ordersTables, err := utils.GetAllExistingShardingTables(context.Background(), "orders")
	if err != nil {
		return fmt.Errorf("list order shards: %w", err)
	}
	for _, tableName := range ordersTables {
		if !facades.Schema().HasTable(tableName) || !facades.Schema().HasColumn(tableName, "expire_at") {
			continue
		}
		_ = facades.Schema().Table(tableName, func(table schema.Blueprint) {
			table.DropIndex("status", "expire_at")
			table.DropColumn("expire_at")
		})
	}
	return nil
}
