package commands

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"

	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/services"
)

type TenantMigrateAll struct{}

func (r *TenantMigrateAll) Signature() string {
	return "tenant:migrate-all"
}

func (r *TenantMigrateAll) Description() string {
	return "对所有启用中的租户执行 migrate（并发上限 2）"
}

func (r *TenantMigrateAll) Extend() command.Extend {
	return command.Extend{Category: "tenant"}
}

func (r *TenantMigrateAll) Handle(ctx console.Context) error {
	var tenants []models.Tenant
	if err := appfacades.PlatformOrmQuery(nil).
		Where("status", models.TenantStatusActive).
		Get(&tenants); err != nil {
		ctx.Error(err.Error())
		return err
	}
	if len(tenants) == 0 {
		ctx.Info("没有启用中的租户")
		return nil
	}

	svc := services.NewTenantConnectionService()
	sem := make(chan struct{}, 2)
	var wg sync.WaitGroup
	var failed atomic.Int64

	for i := range tenants {
		tenant := tenants[i]
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			ctx.Info(fmt.Sprintf("migrate %s ...", tenant.Code))
			if err := svc.MigrateTenant(&tenant); err != nil {
				ctx.Error(fmt.Sprintf("%s 失败: %v", tenant.Code, err))
				failed.Add(1)
				return
			}
			ctx.Success(fmt.Sprintf("%s 完成", tenant.Code))
		}()
	}
	wg.Wait()

	if failed.Load() > 0 {
		return fmt.Errorf("%d tenant migrate(s) failed", failed.Load())
	}
	return nil
}
