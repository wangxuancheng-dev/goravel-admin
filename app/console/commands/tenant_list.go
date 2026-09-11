package commands

import (
	"fmt"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"

	"goravel/app/models"
	"goravel/app/services"
)

type TenantList struct{}

func (r *TenantList) Signature() string { return "tenant:list" }
func (r *TenantList) Description() string {
	return "列出平台库中的租户"
}
func (r *TenantList) Extend() command.Extend {
	return command.Extend{Category: "tenant"}
}

func (r *TenantList) Handle(ctx console.Context) error {
	list, err := services.NewTenantAdminService().ListAll()
	if err != nil {
		ctx.Error(err.Error())
		return err
	}
	if len(list) == 0 {
		ctx.Info("暂无租户")
		return nil
	}
	for _, t := range list {
		status := "active"
		if t.Status == models.TenantStatusDisabled {
			status = "disabled"
		}
		ctx.Info(fmt.Sprintf("id=%d code=%s name=%s status=%s driver=%s isolation=%s db=%s schema=%s",
			t.ID, t.Code, t.Name, status, t.Driver, t.Isolation, t.Database, t.Schema))
	}
	return nil
}
