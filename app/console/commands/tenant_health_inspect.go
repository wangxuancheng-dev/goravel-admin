package commands

import (
	"fmt"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"

	"goravel/app/services"
	"goravel/app/tenancy"
)

// TenantHealthInspect pings tenants and records health/schema/quota issues.
type TenantHealthInspect struct{}

func (r *TenantHealthInspect) Signature() string {
	return "tenant:health-inspect"
}

func (r *TenantHealthInspect) Description() string {
	return "Inspect tenant health (ping/schema/quota/migrate) and optionally alert"
}

func (r *TenantHealthInspect) Extend() command.Extend {
	return command.Extend{
		Category: "tenant",
		Flags: []command.Flag{
			&command.IntFlag{
				Name:  "limit",
				Value: 200,
				Usage: "Max active tenants to inspect",
			},
			&command.BoolFlag{
				Name:  "alert",
				Value: true,
				Usage: "Send webhook/email when warn/fail found",
			},
			&command.BoolFlag{
				Name:  "no-alert",
				Value: false,
				Usage: "Disable alerts for this run",
			},
		},
	}
}

func (r *TenantHealthInspect) Handle(ctx console.Context) error {
	if !tenancy.Enabled() {
		ctx.Warning("tenancy disabled; skip")
		return nil
	}
	limit := ctx.OptionInt("limit")
	alert := ctx.OptionBool("alert")
	if ctx.OptionBool("no-alert") {
		alert = false
	}
	report, err := services.RunTenantHealthInspect(limit, alert)
	if err != nil {
		return err
	}
	if report == nil {
		ctx.Info("no report")
		return nil
	}
	ctx.Info(fmt.Sprintf(
		"health-inspect checked=%d ok=%d warn=%d fail=%d alert=%v",
		report.Checked, report.OK, report.Warn, report.Fail, alert,
	))
	return nil
}
