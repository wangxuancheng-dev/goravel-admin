package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"

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

// Signature The name and signature of the console command.
func (r *CreatePaymentShardingTables) Signature() string {
	return "payment:create-sharding-tables"
}

// Description The console command description.
func (r *CreatePaymentShardingTables) Description() string {

	// # 创建分表（默认创建上个月、当前月份及未来2个月，共4个月）
	// go run . artisan payment:create-sharding-tables

	// # 创建指定月份的分表
	// go run . artisan payment:create-sharding-tables --month=202512

	// # 创建上个月、当前月份及未来N个月的分表（--months 参数包括上个月和当前月）
	// # 例如：--months=6 会创建上个月、当前月及未来4个月，共6个月
	// go run . artisan payment:create-sharding-tables --months=6

	// # 帮助
	// go run . artisan payment:create-sharding-tables --help

	return "创建支付记录分表（按月分表，可指定月份或创建上个月、当前月份及未来几个月）"
}

// Extend The console command extend.
func (r *CreatePaymentShardingTables) Extend() command.Extend {
	return command.Extend{
		Category: "payment",
		Flags: []command.Flag{
			&command.StringFlag{
				Name:    "month",
				Aliases: []string{"m"},
				Usage:   "指定月份(格式: YYYYMM,如:202512),不指定则创建当前月份",
			},
			&command.IntFlag{
				Name:    "months",
				Aliases: []string{"n"},
				Value:   4,
				Usage:   "创建几个月(默认4个月,包括上个月、当前月份及未来2个月)",
			},
		},
	}
}

// Handle Execute the console command.
func (r *CreatePaymentShardingTables) Handle(ctx console.Context) error {
	var months []time.Time
	monthFlag := ctx.Option("month")
	monthsFlag := ctx.OptionInt("months")

	if monthFlag != "" {
		// 指定月份（解析为 UTC 时区，与分表逻辑保持一致）
		parsedTime, err := time.ParseInLocation("200601", monthFlag, time.UTC)
		if err != nil {
			return fmt.Errorf("月份格式错误，应为 YYYYMM 格式（如:202512): %v", err)
		}
		months = []time.Time{parsedTime}
	} else {
		// 创建上个月、当前月份及未来几个月（使用 UTC 时区，与分表逻辑保持一致）
		// 默认：上个月(-1)、当前月(0)、未来2个月(1,2)，共4个月
		now := time.Now().UTC()
		currentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

		// 从 -1 开始（上个月），到 monthsFlag-2（未来几个月）
		// 例如：monthsFlag=4 时，创建：上个月(-1)、当前月(0)、未来1个月(1)、未来2个月(2)
		startOffset := -1 // 从上个月开始
		for i := startOffset; i < monthsFlag-1; i++ {
			month := currentMonth.AddDate(0, i, 0)
			months = append(months, month)
		}
	}

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

	ctx.Info(fmt.Sprintf("\n完成！创建了 %d 个分表，跳过了 %d 个已存在的分表", createdCount, skippedCount))
	return nil
}
