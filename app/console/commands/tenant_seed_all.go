package commands

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"

	"goravel/app/models"
	"goravel/app/services"
	"goravel/app/tenancy"
)

type TenantSeedAll struct{}

func (r *TenantSeedAll) Signature() string { return "tenant:seed-all" }
func (r *TenantSeedAll) Description() string {
	return "Seed all ready tenants (concurrency: TENANCY_MIGRATE_CONCURRENCY / --concurrency)"
}
func (r *TenantSeedAll) Extend() command.Extend {
	return command.Extend{
		Category: "tenant",
		Flags: []command.Flag{
			&command.StringSliceFlag{Name: "class", Aliases: []string{"c"}, Usage: "only run named Seeder(s)"},
			&command.StringSliceFlag{Name: "seeder", Aliases: []string{"s"}, Usage: "same as --class"},
			&command.IntFlag{Name: "concurrency", Usage: "parallel tenants (overrides TENANCY_MIGRATE_CONCURRENCY; clamped 1..100)"},
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
		ctx.Info("no tenants")
		return nil
	}

	ready := make([]models.Tenant, 0, len(list))
	var skipped int
	for _, t := range list {
		if t.Status != models.TenantStatusActive || !t.IsProvisionReady() {
			skipped++
			ctx.Warning(fmt.Sprintf("skip %s (status=%d provision=%s)", t.Code, t.Status, t.ProvisionStatus))
			continue
		}
		ready = append(ready, t)
	}
	if len(ready) == 0 {
		ctx.Info("no ready tenants to seed")
		return nil
	}

	concurrency := tenancy.MigrateConcurrency()
	if raw := strings.TrimSpace(ctx.Option("concurrency")); raw != "" {
		concurrency = tenancy.ClampMigrateConcurrency(ctx.OptionInt("concurrency"))
	}
	ctx.Info(fmt.Sprintf("seed-all: %d tenant(s), concurrency=%d", len(ready), concurrency))

	batchID := services.NewTenantOpsBatchID()
	conn := services.NewTenantConnectionService()
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var failed atomic.Int64

	for i := range ready {
		tenant := ready[i]
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			ctx.Info(fmt.Sprintf("seed %s ...", tenant.Code))
			err := conn.SeedTenant(&tenant, seeders...)
			status := models.TenantOpStatusSuccess
			msg := "seed ok"
			if err != nil {
				status = models.TenantOpStatusFailed
				msg = err.Error()
				ctx.Error(fmt.Sprintf("%s failed: %v", tenant.Code, err))
				failed.Add(1)
			} else {
				ctx.Success(fmt.Sprintf("%s ok", tenant.Code))
			}
			_ = services.RecordDirectTenantOpLog(&tenant, models.TenantOpSeed, status, msg, batchID, services.CliTenantOpActor)
		}()
	}
	wg.Wait()

	if skipped > 0 {
		ctx.Info(fmt.Sprintf("skipped %d non-ready tenant(s)", skipped))
	}
	if failed.Load() > 0 {
		return fmt.Errorf("%d tenant seed(s) failed (batch=%s)", failed.Load(), batchID)
	}
	ctx.Info(fmt.Sprintf("seed-all ok (batch=%s)", batchID))
	return nil
}
