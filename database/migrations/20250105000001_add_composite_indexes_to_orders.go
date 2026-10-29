package migrations

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"

	"goravel/app/utils"
)

type M20250105000001AddCompositeIndexesToOrders struct {
}

func (r *M20250105000001AddCompositeIndexesToOrders) Signature() string {
	return "20250105000001_add_composite_indexes_to_orders"
}

func (r *M20250105000001AddCompositeIndexesToOrders) Up() error {
	// 获取所有已存在的订单分表
	tableNames, err := utils.GetAllExistingShardingTables(context.Background(),"orders")
	if err != nil {
		return fmt.Errorf("获取订单分表失败: %v", err)
	}

	if len(tableNames) == 0 {
		facades.Log().Info("没有找到需要添加索引的分表")
		return nil
	}

	separator := strings.Repeat("=", 60)
	facades.Log().Info(separator)
	facades.Log().Info(fmt.Sprintf("开始为 %d 张订单分表添加复合索引", len(tableNames)))
	facades.Log().Info(separator)

	startTime := time.Now()
	successCount := 0
	skippedCount := 0
	failedTables := []string{}

	// 为每个分表添加复合索引
	for i, tableName := range tableNames {
		progress := fmt.Sprintf("[%d/%d]", i+1, len(tableNames))

		if !facades.Schema().HasTable(tableName) {
			facades.Log().Infof("%s 跳过不存在的分表: %s", progress, tableName)
			skippedCount++
			continue
		}

		tableStartTime := time.Now()

		// 框架生成的索引名称格式：表名_字段1_字段2_..._index
		// 例如：orders_202512_created_at_status_index
		indexName1 := fmt.Sprintf("%s_created_at_status_index", tableName)
		indexName2 := fmt.Sprintf("%s_created_at_user_id_index", tableName)
		indexName3 := fmt.Sprintf("%s_created_at_status_user_id_index", tableName)

		createdCount := 0

		// 检查并创建索引 created_at + status
		if !facades.Schema().HasIndex(tableName, indexName1) {
			if err := facades.Schema().Table(tableName, func(table schema.Blueprint) {
				table.Index("created_at", "status")
			}); err != nil {
				facades.Log().Errorf("%s ✗ 表 %s 创建索引 created_at+status 失败: %v", progress, tableName, err)
				failedTables = append(failedTables, tableName)
				continue
			}
			createdCount++
		}

		// 检查并创建索引 created_at + user_id
		if !facades.Schema().HasIndex(tableName, indexName2) {
			if err := facades.Schema().Table(tableName, func(table schema.Blueprint) {
				table.Index("created_at", "user_id")
			}); err != nil {
				facades.Log().Errorf("%s ✗ 表 %s 创建索引 created_at+user_id 失败: %v", progress, tableName, err)
				failedTables = append(failedTables, tableName)
				continue
			}
			createdCount++
		}

		// 检查并创建索引 created_at + status + user_id
		if !facades.Schema().HasIndex(tableName, indexName3) {
			if err := facades.Schema().Table(tableName, func(table schema.Blueprint) {
				table.Index("created_at", "status", "user_id")
			}); err != nil {
				facades.Log().Errorf("%s ✗ 表 %s 创建索引 created_at+status+user_id 失败: %v", progress, tableName, err)
				failedTables = append(failedTables, tableName)
				continue
			}
			createdCount++
		}

		duration := time.Since(tableStartTime)
		if createdCount == 0 {
			facades.Log().Infof("%s ⊙ 表 %s 索引已存在，跳过 (耗时: %v)", progress, tableName, duration)
			skippedCount++
		} else {
			facades.Log().Infof("%s ✓ 表 %s 创建了 %d 个索引 (耗时: %v)", progress, tableName, createdCount, duration)
			successCount++
		}

		// 每张表之间稍作延迟，避免数据库压力过大
		if i < len(tableNames)-1 {
			time.Sleep(1 * time.Second)
		}
	}

	// 输出最终结果
	totalDuration := time.Since(startTime)
	facades.Log().Info(separator)
	facades.Log().Info(fmt.Sprintf("✅ 完成！总耗时: %v", totalDuration))
	facades.Log().Info(fmt.Sprintf("成功: %d 张表", successCount))
	if skippedCount > 0 {
		facades.Log().Info(fmt.Sprintf("跳过: %d 张表（索引已存在）", skippedCount))
	}
	if len(failedTables) > 0 {
		facades.Log().Warning(fmt.Sprintf("❌ 失败: %d 张表，需要手动处理: %v", len(failedTables), failedTables))
		return fmt.Errorf("部分分表索引创建失败，请检查日志并手动处理")
	}
	facades.Log().Info(separator)

	return nil
}

func (r *M20250105000001AddCompositeIndexesToOrders) Down() error {
	// 获取所有已存在的订单分表
	tableNames, err := utils.GetAllExistingShardingTables(context.Background(),"orders")
	if err != nil {
		return fmt.Errorf("获取订单分表失败: %v", err)
	}

	deletedCount := 0
	failedTables := []string{}

	// 删除索引
	for i, tableName := range tableNames {
		if !facades.Schema().HasTable(tableName) {
			continue
		}

		// 对于复合索引，使用 DropIndex 方法，传入所有字段名
		if err := facades.Schema().Table(tableName, func(table schema.Blueprint) {
			// 删除复合索引：传入所有字段名
			table.DropIndex("created_at", "status", "user_id") // 删除三字段复合索引
			table.DropIndex("created_at", "status")            // 删除两字段复合索引
			table.DropIndex("created_at", "user_id")           // 删除两字段复合索引
		}); err != nil {
			failedTables = append(failedTables, tableName)
			continue
		}
		deletedCount++

		// 每张表之间稍作延迟
		if i < len(tableNames)-1 {
			time.Sleep(1 * time.Second)
		}
	}

	facades.Log().Info(fmt.Sprintf("已删除 %d 张表的复合索引", deletedCount))
	if len(failedTables) > 0 {
		facades.Log().Warning(fmt.Sprintf("❌ 失败: %d 张表，需要手动处理: %v", len(failedTables), failedTables))
		return fmt.Errorf("部分分表索引删除失败，请检查日志并手动处理")
	}

	return nil
}
