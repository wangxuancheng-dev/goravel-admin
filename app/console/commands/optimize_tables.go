package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"

	appfacades "goravel/app/facades"
	"goravel/app/models"
)

type OptimizeTables struct {
}

func (r *OptimizeTables) Signature() string {
	return "db:optimize-tables"
}

func (r *OptimizeTables) Description() string {
	return "优化表（MySQL: OPTIMIZE TABLE; PostgreSQL: VACUUM；tenancy 开启时按租户执行）"
}

func (r *OptimizeTables) Extend() command.Extend {
	return command.Extend{
		Category: "db",
		Flags: []command.Flag{
			&command.StringFlag{
				Name:    "tables",
				Aliases: []string{"t"},
				Usage:   "要优化的表名列表（逗号分隔），也可以直接用参数方式传入多个表名",
			},
			&command.BoolFlag{
				Name:  "full",
				Value: false,
				Usage: "PostgreSQL 是否使用 VACUUM FULL（更重，可能锁表，默认 false）",
			},
			TenantScopeFlag(),
		},
	}
}

func (r *OptimizeTables) Handle(ctx console.Context) error {
	full := ctx.OptionBool("full")

	var tables []string
	tablesFlag := strings.TrimSpace(ctx.Option("tables"))
	if tablesFlag != "" {
		parts := strings.Split(tablesFlag, ",")
		for _, p := range parts {
			t := strings.TrimSpace(p)
			if t != "" {
				tables = append(tables, t)
			}
		}
	}
	for i := 0; ; i++ {
		arg := strings.TrimSpace(ctx.Argument(i))
		if arg == "" {
			break
		}
		tables = append(tables, arg)
	}

	if len(tables) == 0 {
		return fmt.Errorf("请提供要优化的表名，例如：go run . artisan db:optimize-tables payments 或使用 --tables=payments,orders_202601")
	}

	return RunTenantScoped(ctx, func(_ *models.Tenant, bound context.Context) error {
		driver := strings.ToLower(appfacades.OrmQuery(bound).Driver())
		rows := make([][]string, 0, len(tables))

		execOptimize := func(table string) error {
			var sql string
			switch driver {
			case "mysql":
				sql = fmt.Sprintf("OPTIMIZE TABLE `%s`", table)
			case "postgresql":
				if full {
					sql = fmt.Sprintf("VACUUM (FULL, ANALYZE) %s", table)
				} else {
					sql = fmt.Sprintf("VACUUM (ANALYZE) %s", table)
				}
			default:
				return fmt.Errorf("unsupported database driver: %v", driver)
			}
			if _, err := appfacades.OrmQuery(bound).Exec(sql); err != nil {
				return err
			}
			rows = append(rows, []string{table, "成功"})
			return nil
		}

		ctx.Info("开始执行优化...")
		for _, table := range tables {
			if appfacades.SchemaHasTable(bound, table) {
				if err := execOptimize(table); err != nil {
					return fmt.Errorf("optimize %s failed: %v", table, err)
				}
			} else {
				rows = append(rows, []string{table, "跳过（表不存在）"})
			}
		}
		if len(rows) > 0 {
			ctx.Table([]string{"表名", "状态"}, rows)
		}
		ctx.Info("完成")
		return nil
	})
}
