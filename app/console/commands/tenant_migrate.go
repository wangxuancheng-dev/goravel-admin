package commands

import (
	"fmt"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"

	"goravel/app/models"
	"goravel/app/services"
)

type TenantMigrate struct{}

func (r *TenantMigrate) Signature() string {
	return "tenant:migrate"
}

func (r *TenantMigrate) Description() string {
	return "在指定租户连接上执行 migrate"
}

func (r *TenantMigrate) Extend() command.Extend {
	return command.Extend{Category: "tenant"}
}

func (r *TenantMigrate) Handle(ctx console.Context) error {
	idOrCode := ctx.Argument(0)
	if idOrCode == "" {
		ctx.Error("用法: tenant:migrate {id|code}")
		return nil
	}

	svc := services.NewTenantConnectionService()
	tenant, err := svc.FindTenantByIDOrCode(idOrCode)
	if err != nil {
		ctx.Error("租户不存在: " + idOrCode)
		return nil
	}

	ctx.Info(fmt.Sprintf("正在 migrate 租户 %s (%s)...", tenant.Code, tenant.ConnectionName))
	err = svc.MigrateTenant(tenant)
	status := models.TenantOpStatusSuccess
	msg := "migrate ok"
	if err != nil {
		status = models.TenantOpStatusFailed
		msg = err.Error()
		_ = services.RecordDirectTenantOpLog(tenant, models.TenantOpMigrate, status, msg, "", services.CliTenantOpActor)
		ctx.Error(err.Error())
		return err
	}
	_ = services.RecordDirectTenantOpLog(tenant, models.TenantOpMigrate, status, msg, "", services.CliTenantOpActor)
	ctx.Success("migrate 完成")
	return nil
}
