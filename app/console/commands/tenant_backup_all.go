package commands

import (
	"fmt"
	"strings"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"
	"github.com/goravel/framework/facades"

	"goravel/app/models"
	"goravel/app/services"
	"goravel/app/tenancy"
)

type TenantBackupAll struct{}

func (r *TenantBackupAll) Signature() string { return "tenant:backup-all" }
func (r *TenantBackupAll) Description() string {
	return "备份所有启用且已 ready 的租户（逐个调用 tenant:backup）"
}
func (r *TenantBackupAll) Extend() command.Extend {
	return command.Extend{
		Category: "tenant",
		Flags: []command.Flag{
			&command.IntFlag{Name: "keep", Usage: "透传给 tenant:backup 的保留份数"},
		},
	}
}

func (r *TenantBackupAll) Handle(ctx console.Context) error {
	if !tenancy.Enabled() {
		ctx.Error("需要 TENANCY_DRIVER=database")
		return nil
	}
	list, err := services.NewTenantAdminService().ListAll()
	if err != nil {
		ctx.Error(err.Error())
		return err
	}
	failed := 0
	for i := range list {
		t := list[i]
		if t.Status != models.TenantStatusActive || !t.IsProvisionReady() {
			continue
		}
		cmd := "tenant:backup " + t.Code
		if raw := strings.TrimSpace(ctx.Option("keep")); raw != "" {
			cmd += " --keep=" + raw
		}
		ctx.Info(fmt.Sprintf("backup %s ...", t.Code))
		if err := facades.Artisan().Call(cmd); err != nil {
			ctx.Error(fmt.Sprintf("%s: %v", t.Code, err))
			failed++
		}
	}
	if failed > 0 {
		return fmt.Errorf("%d tenant backup(s) failed", failed)
	}
	ctx.Success("tenant:backup-all 完成")
	return nil
}
