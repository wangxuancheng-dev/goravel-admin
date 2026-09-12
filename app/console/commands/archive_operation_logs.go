package commands

import (
	"context"
	"fmt"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"

	"goravel/app/constants"
	"goravel/app/models"
	"goravel/app/services"
)

type ArchiveOperationLogs struct {
}

func (r *ArchiveOperationLogs) Signature() string {
	return "operation_log:archive"
}

func (r *ArchiveOperationLogs) Description() string {
	return "归档并清理过期操作日志（先导出 CSV 再删除；tenancy 开启时按租户执行）"
}

func (r *ArchiveOperationLogs) Extend() command.Extend {
	return command.Extend{
		Category: "operation_log",
		Flags: []command.Flag{
			TenantScopeFlag(),
			&command.IntFlag{
				Name:    "days",
				Aliases: []string{"d"},
				Value:   constants.DefaultCleanLogDays,
				Usage:   "归档多少天前的操作日志（默认 30）",
			},
		},
	}
}

func (r *ArchiveOperationLogs) Handle(ctx console.Context) error {
	days := ctx.OptionInt("days")
	if days <= 0 {
		days = constants.DefaultCleanLogDays
	}

	return RunTenantScoped(ctx, func(_ *models.Tenant, bound context.Context) error {
		ctx.Info(fmt.Sprintf("开始归档 %d 天前的操作日志...", days))
		exportID, err := services.NewOperationLogService(bound).Archive(days)
		if err != nil {
			return fmt.Errorf("归档操作日志失败: %w", err)
		}
		ctx.Info(fmt.Sprintf("操作日志归档完成，export_id=%d", exportID))
		return nil
	})
}
