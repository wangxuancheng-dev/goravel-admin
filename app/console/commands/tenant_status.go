package commands

import (
	"fmt"
	"strconv"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"

	"goravel/app/models"
	"goravel/app/services"
)

type TenantDisable struct{}

func (r *TenantDisable) Signature() string { return "tenant:disable" }
func (r *TenantDisable) Description() string {
	return "禁用租户（id 或 code）"
}
func (r *TenantDisable) Extend() command.Extend {
	return command.Extend{Category: "tenant"}
}

func (r *TenantDisable) Handle(ctx console.Context) error {
	return setTenantStatus(ctx, models.TenantStatusDisabled)
}

type TenantEnable struct{}

func (r *TenantEnable) Signature() string { return "tenant:enable" }
func (r *TenantEnable) Description() string {
	return "启用租户（id 或 code）"
}
func (r *TenantEnable) Extend() command.Extend {
	return command.Extend{Category: "tenant"}
}

func (r *TenantEnable) Handle(ctx console.Context) error {
	return setTenantStatus(ctx, models.TenantStatusActive)
}

func setTenantStatus(ctx console.Context, status uint8) error {
	raw := ctx.Argument(0)
	if raw == "" {
		ctx.Error("用法: tenant:disable|enable {id|code}")
		return nil
	}
	admin := services.NewTenantAdminService()
	conn := services.NewTenantConnectionService()
	tenant, err := conn.FindTenantByIDOrCode(raw)
	if err != nil {
		ctx.Error("租户不存在: " + raw)
		return nil
	}
	updated, err := admin.SetStatus(tenant.ID, status)
	if err != nil {
		ctx.Error(err.Error())
		return err
	}
	label := "disabled"
	if status == models.TenantStatusActive {
		label = "enabled"
	}
	ctx.Success(fmt.Sprintf("tenant %s (%s) %s", updated.Code, strconv.FormatUint(uint64(updated.ID), 10), label))
	return nil
}
