package commands

import (
	"fmt"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"

	"goravel/app/models"
	"goravel/app/services"
)

type TenantRollback struct{}

func (r *TenantRollback) Signature() string { return "tenant:rollback" }
func (r *TenantRollback) Description() string {
	return "Rollback tenant migrations (default: last batch; --step=N or --batch=N)"
}
func (r *TenantRollback) Extend() command.Extend {
	return command.Extend{
		Category: "tenant",
		Flags: []command.Flag{
			&command.IntFlag{Name: "step", Usage: "rollback last N migration files (takes precedence over --batch when >0)"},
			&command.IntFlag{Name: "batch", Usage: "rollback migrations.batch = N (0 with step=0 = last batch)"},
		},
	}
}

func (r *TenantRollback) Handle(ctx console.Context) error {
	idOrCode := ctx.Argument(0)
	if idOrCode == "" {
		ctx.Error("usage: tenant:rollback {id|code} [--step=N] [--batch=N]")
		return nil
	}
	svc := services.NewTenantConnectionService()
	tenant, err := svc.FindTenantByIDOrCode(idOrCode)
	if err != nil {
		ctx.Error("tenant not found: " + idOrCode)
		return nil
	}
	step := ctx.OptionInt("step")
	batch := ctx.OptionInt("batch")
	ctx.Info(fmt.Sprintf("rollback tenant %s (step=%d batch=%d)...", tenant.Code, step, batch))
	err = svc.RollbackTenant(tenant, step, batch)
	status := models.TenantOpStatusSuccess
	msg := fmt.Sprintf("rollback ok (step=%d batch=%d)", step, batch)
	if err != nil {
		status = models.TenantOpStatusFailed
		msg = err.Error()
		_ = services.RecordDirectTenantOpLog(tenant, models.TenantOpRollback, status, msg, "", services.CliTenantOpActor)
		ctx.Error(err.Error())
		return err
	}
	_ = services.RecordDirectTenantOpLog(tenant, models.TenantOpRollback, status, msg, "", services.CliTenantOpActor)
	ctx.Success("rollback done")
	return nil
}
