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

type TenantRollbackAll struct{}

func (r *TenantRollbackAll) Signature() string { return "tenant:rollback-all" }
func (r *TenantRollbackAll) Description() string {
	return "Rollback migrations on all active tenants (TENANCY_MIGRATE_CONCURRENCY / --concurrency)"
}
func (r *TenantRollbackAll) Extend() command.Extend {
	return command.Extend{
		Category: "tenant",
		Flags: []command.Flag{
			&command.IntFlag{Name: "step", Usage: "rollback last N migration files"},
			&command.IntFlag{Name: "batch", Usage: "rollback migrations.batch = N (0/0 = last batch)"},
			&command.IntFlag{Name: "concurrency", Usage: "parallel tenants (overrides TENANCY_MIGRATE_CONCURRENCY; 1..100)"},
		},
	}
}

func (r *TenantRollbackAll) Handle(ctx console.Context) error {
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

	step := ctx.OptionInt("step")
	batch := ctx.OptionInt("batch")
	concurrency := tenancy.MigrateConcurrency()
	if raw := strings.TrimSpace(ctx.Option("concurrency")); raw != "" {
		concurrency = tenancy.ClampMigrateConcurrency(ctx.OptionInt("concurrency"))
	}

	ctx.Info(fmt.Sprintf("rollback-all: %d tenant(s), concurrency=%d step=%d batch=%d",
		len(tenants), concurrency, step, batch))

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

			ctx.Info(fmt.Sprintf("rollback %s ...", tenant.Code))
			err := svc.RollbackTenant(&tenant, step, batch)
			status := models.TenantOpStatusSuccess
			msg := fmt.Sprintf("rollback ok (step=%d batch=%d)", step, batch)
			if err != nil {
				status = models.TenantOpStatusFailed
				msg = err.Error()
				ctx.Error(fmt.Sprintf("%s failed: %v", tenant.Code, err))
				failed.Add(1)
			} else {
				ctx.Success(fmt.Sprintf("%s done", tenant.Code))
			}
			_ = services.RecordDirectTenantOpLog(&tenant, models.TenantOpRollback, status, msg, batchID, services.CliTenantOpActor)
		}()
	}
	wg.Wait()

	if failed.Load() > 0 {
		return fmt.Errorf("%d tenant rollback(s) failed (batch=%s)", failed.Load(), batchID)
	}
	ctx.Info(fmt.Sprintf("rollback-all ok (batch=%s)", batchID))
	return nil
}
