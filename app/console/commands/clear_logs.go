package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"

	appfacades "goravel/app/facades"
	"goravel/app/models"
)

type ClearLogs struct {
}

func (r *ClearLogs) Signature() string {
	return "app:clear-logs"
}

func (r *ClearLogs) Description() string {
	return "清理6个月前的日志记录（操作日志、登录日志、系统日志；tenancy 开启时按租户执行）"
}

func (r *ClearLogs) Extend() command.Extend {
	return command.Extend{
		Category: "app",
		Flags:    []command.Flag{TenantScopeFlag()},
	}
}

func (r *ClearLogs) Handle(ctx console.Context) error {
	monthsAgo := time.Now().AddDate(0, -6, 0)

	return RunTenantScoped(ctx, func(_ *models.Tenant, bound context.Context) error {
		ctx.Info("开始清理6个月前的日志...")

		operationLogResult, err := appfacades.OrmQuery(bound).Model(&models.OperationLog{}).
			Where("created_at < ?", monthsAgo).
			Delete(&models.OperationLog{})
		if err != nil {
			return fmt.Errorf("清理操作日志失败: %w", err)
		}
		ctx.Info(fmt.Sprintf("已清理操作日志: %d 条", operationLogResult.RowsAffected))

		loginLogResult, err := appfacades.OrmQuery(bound).Model(&models.LoginLog{}).
			Where("created_at < ?", monthsAgo).
			Delete(&models.LoginLog{})
		if err != nil {
			return fmt.Errorf("清理登录日志失败: %w", err)
		}
		ctx.Info(fmt.Sprintf("已清理登录日志: %d 条", loginLogResult.RowsAffected))

		systemLogResult, err := appfacades.OrmQuery(bound).Model(&models.SystemLog{}).
			Where("created_at < ?", monthsAgo).
			Delete(&models.SystemLog{})
		if err != nil {
			return fmt.Errorf("清理系统日志失败: %w", err)
		}
		ctx.Info(fmt.Sprintf("已清理系统日志: %d 条", systemLogResult.RowsAffected))

		ctx.Info("日志清理完成！")
		return nil
	})
}
