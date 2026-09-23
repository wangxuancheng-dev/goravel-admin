package commands

import (
	"fmt"
	"strings"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"

	"goravel/app/models"
	"goravel/app/services"
)

type TenantSeed struct{}

func (r *TenantSeed) Signature() string {
	return "tenant:seed"
}

func (r *TenantSeed) Description() string {
	return "在指定租户库执行 db:seed（可 --class/--seeder 指定单个 Seeder）"
}

func (r *TenantSeed) Extend() command.Extend {
	return command.Extend{
		Category: "tenant",
		Flags: []command.Flag{
			&command.StringSliceFlag{
				Name:    "class",
				Aliases: []string{"c"},
				Usage:   "只跑指定 Seeder（可多次），如 MenuSeeder；与 --seeder 等价",
			},
			&command.StringSliceFlag{
				Name:    "seeder",
				Aliases: []string{"s"},
				Usage:   "同 --class，对齐 db:seed --seeder",
			},
		},
	}
}

func (r *TenantSeed) Handle(ctx console.Context) error {
	idOrCode := ctx.Argument(0)
	if idOrCode == "" {
		ctx.Error("用法: tenant:seed {id|code} [--class=MenuSeeder] [--seeder=PermissionSeeder]")
		return nil
	}
	svc := services.NewTenantConnectionService()
	tenant, err := svc.FindTenantByIDOrCode(idOrCode)
	if err != nil {
		ctx.Error("租户不存在: " + idOrCode)
		return nil
	}

	seeders := mergeSeederNames(ctx.OptionSlice("class"), ctx.OptionSlice("seeder"))
	if len(seeders) == 0 {
		ctx.Info(fmt.Sprintf("正在 seed 租户 %s（全部）...", tenant.Code))
	} else {
		ctx.Info(fmt.Sprintf("正在 seed 租户 %s（%s）...", tenant.Code, strings.Join(seeders, ", ")))
	}

	if err := svc.SeedTenant(tenant, seeders...); err != nil {
		_ = services.RecordDirectTenantOpLog(tenant, models.TenantOpSeed, models.TenantOpStatusFailed, err.Error(), "", services.CliTenantOpActor)
		ctx.Error(err.Error())
		return err
	}
	_ = services.RecordDirectTenantOpLog(tenant, models.TenantOpSeed, models.TenantOpStatusSuccess, "seed ok", "", services.CliTenantOpActor)
	ctx.Success("seed 完成")
	return nil
}

func mergeSeederNames(groups ...[]string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, group := range groups {
		for _, name := range group {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			out = append(out, name)
		}
	}
	return out
}
