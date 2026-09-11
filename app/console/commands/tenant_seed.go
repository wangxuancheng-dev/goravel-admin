package commands

import (
	"fmt"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"

	"goravel/app/services"
)

type TenantSeed struct{}

func (r *TenantSeed) Signature() string {
	return "tenant:seed"
}

func (r *TenantSeed) Description() string {
	return "在指定租户库执行 db:seed（RBAC/管理员等）"
}

func (r *TenantSeed) Extend() command.Extend {
	return command.Extend{Category: "tenant"}
}

func (r *TenantSeed) Handle(ctx console.Context) error {
	idOrCode := ctx.Argument(0)
	if idOrCode == "" {
		ctx.Error("用法: tenant:seed {id|code}")
		return nil
	}
	svc := services.NewTenantConnectionService()
	tenant, err := svc.FindTenantByIDOrCode(idOrCode)
	if err != nil {
		ctx.Error("租户不存在: " + idOrCode)
		return nil
	}
	ctx.Info(fmt.Sprintf("正在 seed 租户 %s ...", tenant.Code))
	if err := svc.SeedTenant(tenant); err != nil {
		ctx.Error(err.Error())
		return err
	}
	ctx.Success("seed 完成")
	return nil
}
