package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"

	"goravel/app/models"
	"goravel/app/services"
	"goravel/app/utils"
	"goravel/app/utils/errorlog"
)

type CreateOrderShardingTables struct {
	shardingService services.ShardingService
}

func NewCreateOrderShardingTables() *CreateOrderShardingTables {
	return &CreateOrderShardingTables{
		shardingService: services.NewShardingService(context.Background()),
	}
}

func (r *CreateOrderShardingTables) Signature() string {
	return "order:create-sharding-tables"
}

func (r *CreateOrderShardingTables) Description() string {
	return "创建订单分表（按月分表；tenancy 开启时按租户执行）"
}

func (r *CreateOrderShardingTables) Extend() command.Extend {
	return command.Extend{
		Category: "order",
		Flags: []command.Flag{
			&command.StringFlag{
				Name:    "month",
				Aliases: []string{"m"},
				Usage:   "指定月份(格式: YYYYMM,如:202512),不指定则按 --months 创建",
			},
			&command.IntFlag{
				Name:    "months",
				Aliases: []string{"n"},
				Value:   4,
				Usage:   "创建几个月(默认4个月,包括上个月、当前月份及未来2个月)",
			},
			TenantScopeFlag(),
		},
	}
}

func (r *CreateOrderShardingTables) Handle(ctx console.Context) error {
	months, err := resolveShardingMonths(ctx)
	if err != nil {
		return err
	}

	return RunTenantScoped(ctx, func(_ *models.Tenant, _ context.Context) error {
		createdCount := 0
		skippedCount := 0

		for _, month := range months {
			tableName := utils.GetShardingTableName("orders", month)
			detailTableName := utils.GetShardingTableName("order_details", month)

			if utils.ShardingTableExists(tableName) {
				ctx.Info(fmt.Sprintf("分表 %s 已存在，跳过", tableName))
				skippedCount++
			} else {
				if err := r.shardingService.CreateShardingTable(tableName, "orders"); err != nil {
					errorlog.Record(context.Background(), "sharding", "创建分表失败", map[string]any{
						"table_name":      tableName,
						"base_table_name": "orders",
						"error":           err.Error(),
					}, "创建分表 %s 失败: %v", tableName, err)
					return fmt.Errorf("创建分表 %s 失败: %v", tableName, err)
				}
				utils.MarkShardingTableExists(tableName)
				ctx.Info(fmt.Sprintf("✓ 创建分表: %s", tableName))
				createdCount++
			}

			if utils.ShardingTableExists(detailTableName) {
				ctx.Info(fmt.Sprintf("分表 %s 已存在，跳过", detailTableName))
			} else {
				if err := r.shardingService.CreateShardingTable(detailTableName, "order_details"); err != nil {
					errorlog.Record(context.Background(), "sharding", "创建分表失败", map[string]any{
						"table_name":      detailTableName,
						"base_table_name": "order_details",
						"error":           err.Error(),
					}, "创建分表 %s 失败: %v", detailTableName, err)
					return fmt.Errorf("创建分表 %s 失败: %v", detailTableName, err)
				}
				utils.MarkShardingTableExists(detailTableName)
				ctx.Info(fmt.Sprintf("✓ 创建分表: %s", detailTableName))
			}
		}

		ctx.Info(fmt.Sprintf("完成！创建了 %d 个分表，跳过了 %d 个已存在的分表", createdCount, skippedCount))
		return nil
	})
}

func resolveShardingMonths(ctx console.Context) ([]time.Time, error) {
	monthFlag := ctx.Option("month")
	monthsFlag := ctx.OptionInt("months")
	if monthFlag != "" {
		parsedTime, err := time.ParseInLocation("200601", monthFlag, time.UTC)
		if err != nil {
			return nil, fmt.Errorf("月份格式错误，应为 YYYYMM 格式（如:202512): %v", err)
		}
		return []time.Time{parsedTime}, nil
	}

	now := time.Now().UTC()
	currentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	months := make([]time.Time, 0, monthsFlag)
	startOffset := -1
	for i := startOffset; i < monthsFlag-1; i++ {
		months = append(months, currentMonth.AddDate(0, i, 0))
	}
	return months, nil
}
