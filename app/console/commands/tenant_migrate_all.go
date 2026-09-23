package commands

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"

	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/app/tenancy"
)

type TenantMigrateAll struct{}

func (r *TenantMigrateAll) Signature() string {
	return "tenant:migrate-all"
}

func (r *TenantMigrateAll) Description() string {
	return "Migrate all active tenants (concurrency: TENANCY_MIGRATE_CONCURRENCY / --concurrency)"
}

func (r *TenantMigrateAll) Extend() command.Extend {
	return command.Extend{
		Category: "tenant",
		Flags: []command.Flag{
			&command.IntFlag{
				Name:  "concurrency",
				Usage: "parallel tenants (overrides TENANCY_MIGRATE_CONCURRENCY; clamped 1..100)",
			},
		},
	}
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
		ctx.Info("no active tenants")
		return nil
	}

	concurrency := tenancy.MigrateConcurrency()
	if raw := strings.TrimSpace(ctx.Option("concurrency")); raw != "" {
		concurrency = tenancy.ClampMigrateConcurrency(ctx.OptionInt("concurrency"))
	}

	ctx.Info(fmt.Sprintf("migrate-all: %d tenant(s), concurrency=%d", len(tenants), concurrency))

	batchID := services.NewTenantOpsBatchID()
	svc := services.NewTenantConnectionService()
	sem := make(chan struct{}, concurrency)
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
			err := svc.MigrateTenant(&tenant)
			status := models.TenantOpStatusSuccess
			msg := "migrate ok"
			if err != nil {
				status = models.TenantOpStatusFailed
				msg = err.Error()
				ctx.Error(fmt.Sprintf("%s failed: %v", tenant.Code, err))
				failed.Add(1)
			} else {
				ctx.Success(fmt.Sprintf("%s done", tenant.Code))
			}
			_ = services.RecordDirectTenantOpLog(&tenant, models.TenantOpMigrate, status, msg, batchID, services.CliTenantOpActor)
		}()
	}
	wg.Wait()

	if failed.Load() > 0 {
		return fmt.Errorf("%d tenant migrate(s) failed (batch=%s)", failed.Load(), batchID)
	}
	ctx.Info(fmt.Sprintf("migrate-all ok (batch=%s)", batchID))
	return nil
}
