package commands

import (
	"fmt"
	"strings"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"
	"github.com/goravel/framework/facades"

	"goravel/app/services"
	"goravel/app/tenancy"
)

type TenantBackup struct{}

func (r *TenantBackup) Signature() string { return "tenant:backup" }
func (r *TenantBackup) Description() string {
	return "备份指定租户数据库到 storage/backups/tenants/{code}/（依赖 mysqldump/pg_dump）"
}
func (r *TenantBackup) Extend() command.Extend {
	return command.Extend{
		Category: "tenant",
		Flags: []command.Flag{
			&command.IntFlag{Name: "keep", Usage: "保留最近 N 份备份（0=不清理；默认读 TENANCY_BACKUP_KEEP）"},
		},
	}
}

func (r *TenantBackup) Handle(ctx console.Context) error {
	if !tenancy.Enabled() {
		ctx.Error("需要 TENANCY_DRIVER=database")
		return nil
	}
	idOrCode := ctx.Argument(0)
	if idOrCode == "" {
		ctx.Error("用法: tenant:backup {id|code} [--keep=N]")
		return nil
	}
	tenant, err := services.NewTenantConnectionService().FindTenantByIDOrCode(idOrCode)
	if err != nil {
		ctx.Error("租户不存在: " + idOrCode)
		return nil
	}
	keep := -1
	if raw := strings.TrimSpace(ctx.Option("keep")); raw != "" {
		keep = ctx.OptionInt("keep")
	} else {
		keep = facades.Config().GetInt("tenancy.backup_keep", 10)
	}
	outFile, err := services.BackupTenant(tenant, keep)
	if err != nil {
		ctx.Error(err.Error())
		return err
	}
	_ = services.MarkTenantBackupResult(tenant, outFile)
	ctx.Success("备份完成: " + outFile)
	if keep > 0 {
		ctx.Info(fmt.Sprintf("保留策略: 最近 %d 份", keep))
	}
	return nil
}
