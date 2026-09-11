package commands

import (
	"context"
	"fmt"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"

	"goravel/app/models"
	"goravel/app/services"
	"goravel/app/utils"
	"goravel/app/utils/errorlog"
)

type CreatePaymentShardingTables struct {
	shardingService services.ShardingService
}

func NewCreatePaymentShardingTables() *CreatePaymentShardingTables {
	return &CreatePaymentShardingTables{
		shardingService: services.NewShardingService(context.Background()),
	}
}

func (r *CreatePaymentShardingTables) Signature() string {
	return "payment:create-sharding-tables"
}

func (r *CreatePaymentShardingTables) Description() string {
	return "创建支付记录分表（按月分表；tenancy 开启时按租户执行）"
}

func (r *CreatePaymentShardingTables) Extend() command.Extend {
	return command.Extend{
		Category: "payment",
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

func (r *CreatePaymentShardingTables) Handle(ctx console.Context) error {
	months, err := resolveShardingMonths(ctx)
	if err != nil {
		return err
	}

	return RunTenantScoped(ctx, func(_ *models.Tenant, _ context.Context) error {
		createdCount := 0
		skippedCount := 0

		for _, month := range months {
			tableName := utils.GetShardingTableName("payments", month)
			if utils.ShardingTableExists(tableName) {
				ctx.Info(fmt.Sprintf("分表 %s 已存在，跳过", tableName))
				skippedCount++
				continue
			}
			if err := r.shardingService.CreateShardingTable(tableName, "payments"); err != nil {
				errorlog.Record(context.Background(), "sharding", "创建支付记录分表失败", map[string]any{
					"table_name": tableName,
					"error":      err.Error(),
				}, "创建支付记录分表 %s 失败: %v", tableName, err)
				return fmt.Errorf("创建支付记录分表 %s 失败: %v", tableName, err)
			}
			utils.MarkShardingTableExists(tableName)
			ctx.Info(fmt.Sprintf("✓ 创建分表: %s", tableName))
			createdCount++
		}

		ctx.Info(fmt.Sprintf("完成！创建了 %d 个分表，跳过了 %d 个已存在的分表", createdCount, skippedCount))
		return nil
	})
}
