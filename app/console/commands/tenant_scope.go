package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"

	"goravel/app/models"
	"goravel/app/services"
	"goravel/app/tenancy"
)

// TenantScopeFlag returns the shared --tenant option for ops commands.
func TenantScopeFlag() command.Flag {
	return &command.StringFlag{
		Name:    "tenant",
		Aliases: []string{"T"},
		Usage:   "租户 code 或 id；TENANCY_DRIVER=database 且省略时遍历所有启用租户",
	}
}

// RunTenantScoped executes work on one tenant, all active tenants, or the default DB when tenancy is off.
func RunTenantScoped(ctx console.Context, work services.TenantWork) error {
	svc := services.NewTenantConnectionService()
	hint := strings.TrimSpace(ctx.Option("tenant"))
	if tenancy.Enabled() && hint == "" {
		ctx.Info("tenancy 已开启：将按启用中的租户逐个执行（可用 --tenant 限定）")
	}
	return svc.RunTenantScope(hint, func(tenant *models.Tenant, bound context.Context) error {
		if tenant != nil {
			ctx.Info(fmt.Sprintf("[%s] ...", tenant.Code))
		}
		if err := work(tenant, bound); err != nil {
			if tenant != nil {
				ctx.Error(fmt.Sprintf("[%s] %v", tenant.Code, err))
			}
			return err
		}
		if tenant != nil {
			ctx.Success(fmt.Sprintf("[%s] 完成", tenant.Code))
		}
		return nil
	})
}

// RunTenantScopedRequire requires --tenant when tenancy is on (for write-heavy commands).
func RunTenantScopedRequire(ctx console.Context, work services.TenantWork) error {
	if tenancy.Enabled() && strings.TrimSpace(ctx.Option("tenant")) == "" {
		return fmt.Errorf("tenancy 已开启：请指定 --tenant={code|id}（避免对所有租户执行写操作）")
	}
	return RunTenantScoped(ctx, work)
}
