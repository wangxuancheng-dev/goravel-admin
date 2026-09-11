package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"

	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/utils"
)

type AnalyzeStats struct {
}

func (r *AnalyzeStats) Signature() string {
	return "db:analyze-stats"
}

func (r *AnalyzeStats) Description() string {
	return "更新订单分表与支付表统计信息（ANALYZE；tenancy 开启时按租户执行）"
}

func (r *AnalyzeStats) Extend() command.Extend {
	return command.Extend{
		Category: "db",
		Flags: []command.Flag{
			&command.StringFlag{
				Name:    "month",
				Aliases: []string{"m"},
				Usage:   "指定月份(格式: YYYYMM,如:202512)，不指定则按 months 向前分析",
			},
			&command.IntFlag{
				Name:    "months",
				Aliases: []string{"n"},
				Value:   2,
				Usage:   "向前分析几个月（默认2：当前月+上个月）",
			},
			&command.BoolFlag{
				Name:  "orders",
				Value: true,
				Usage: "是否分析订单分表（orders_YYYYMM）",
			},
			&command.BoolFlag{
				Name:  "order-details",
				Value: true,
				Usage: "是否分析订单详情分表（order_details_YYYYMM）",
			},
			&command.BoolFlag{
				Name:  "payments",
				Value: true,
				Usage: "是否分析支付记录表（payments）",
			},
			TenantScopeFlag(),
		},
	}
}

func (r *AnalyzeStats) Handle(ctx console.Context) error {
	monthsFlag := ctx.OptionInt("months")
	if monthsFlag <= 0 {
		monthsFlag = 2
	}

	monthFlag := strings.TrimSpace(ctx.Option("month"))
	analyzeOrders := ctx.OptionBool("orders")
	analyzeOrderDetails := ctx.OptionBool("order-details")
	analyzePayments := ctx.OptionBool("payments")

	var months []time.Time
	if monthFlag != "" {
		parsedTime, err := time.ParseInLocation("200601", monthFlag, time.UTC)
		if err != nil {
			return fmt.Errorf("月份格式错误，应为 YYYYMM 格式（如:202512): %v", err)
		}
		months = []time.Time{parsedTime}
	} else {
		now := time.Now().UTC()
		currentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		for i := 0; i < monthsFlag; i++ {
			months = append(months, currentMonth.AddDate(0, -i, 0))
		}
	}

	return RunTenantScoped(ctx, func(_ *models.Tenant, bound context.Context) error {
		driver := strings.ToLower(appfacades.OrmQuery(bound).Driver())
		ctx.Info("开始执行 ANALYZE...")

		rows := make([][]string, 0, 16)
		execAnalyze := func(table string) error {
			var sql string
			switch driver {
			case "mysql":
				sql = fmt.Sprintf("ANALYZE TABLE `%s`", table)
			case "postgresql":
				sql = fmt.Sprintf("ANALYZE %s", table)
			default:
				return fmt.Errorf("unsupported database driver: %v", driver)
			}
			if _, err := appfacades.OrmQuery(bound).Exec(sql); err != nil {
				return err
			}
			rows = append(rows, []string{table, "成功"})
			return nil
		}

		if analyzeOrders {
			for _, m := range months {
				table := utils.GetShardingTableName("orders", m)
				if utils.ShardingTableExistsCtx(bound, table) {
					if err := execAnalyze(table); err != nil {
						return fmt.Errorf("analyze %s failed: %v", table, err)
					}
				}
			}
		}

		if analyzeOrderDetails {
			for _, m := range months {
				table := utils.GetShardingTableName("order_details", m)
				if utils.ShardingTableExistsCtx(bound, table) {
					if err := execAnalyze(table); err != nil {
						return fmt.Errorf("analyze %s failed: %v", table, err)
					}
				}
			}
		}

		if analyzePayments {
			if utils.ShardingTableExistsCtx(bound, "payments") || appfacades.SchemaHasTable(bound, "payments") {
				if err := execAnalyze("payments"); err != nil {
					return fmt.Errorf("analyze payments failed: %v", err)
				}
			}
		}

		if len(rows) > 0 {
			ctx.Table([]string{"表名", "状态"}, rows)
		}
		ctx.Info("完成")
		return nil
	})
}
