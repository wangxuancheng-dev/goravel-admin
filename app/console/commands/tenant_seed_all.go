package commands

import (
	"fmt"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"

	"goravel/app/services"
)

type TenantSeedAll struct{}

func (r *TenantSeedAll) Signature() string { return "tenant:seed-all" }
func (r *TenantSeedAll) Description() string {
	return "对所有租户执行 db:seed（可 --class/--seeder）"
}
func (r *TenantSeedAll) Extend() command.Extend {
	return command.Extend{
		Category: "tenant",
		Flags: []command.Flag{
			&command.StringSliceFlag{Name: "class", Aliases: []string{"c"}, Usage: "只跑指定 Seeder"},
			&command.StringSliceFlag{Name: "seeder", Aliases: []string{"s"}, Usage: "同 --class"},
		},
	}
}

func (r *TenantSeedAll) Handle(ctx console.Context) error {
	seeders := mergeSeederNames(ctx.OptionSlice("class"), ctx.OptionSlice("seeder"))
	list, err := services.NewTenantAdminService().ListAll()
	if err != nil {
		ctx.Error(err.Error())
		return err
	}
	if len(list) == 0 {
		ctx.Info("暂无租户")
		return nil
	}
	conn := services.NewTenantConnectionService()
	var failed int
	for _, t := range list {
		tenant := t
		ctx.Info(fmt.Sprintf("seed %s ...", tenant.Code))
		if err := conn.SeedTenant(&tenant, seeders...); err != nil {
			ctx.Error(fmt.Sprintf("%s: %v", tenant.Code, err))
			failed++
			continue
		}
		ctx.Success(fmt.Sprintf("%s ok", tenant.Code))
	}
	if failed > 0 {
		return fmt.Errorf("%d tenant seed(s) failed", failed)
	}
	return nil
}
