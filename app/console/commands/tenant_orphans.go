package commands

import (
	"fmt"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"

	"goravel/app/services"
	"goravel/app/tenancy"
)

// TenantOrphans lists (and with --drop removes) tenant databases / schemas that no
// tenant row references: what permanently deleted tenants left behind before deletes
// erased their storage. Such a leftover blocks a new tenant with the same code.
type TenantOrphans struct{}

func (r *TenantOrphans) Signature() string {
	return "tenant:orphans"
}

func (r *TenantOrphans) Description() string {
	return "List tenant databases/schemas no tenant references (--drop to remove them)"
}

func (r *TenantOrphans) Extend() command.Extend {
	return command.Extend{
		Category: "tenant",
		Flags: []command.Flag{
			&command.BoolFlag{
				Name:  "drop",
				Usage: "drop the listed databases / schemas and their dedicated accounts (irreversible; back up first)",
			},
			&command.BoolFlag{
				Name:  "yes",
				Usage: "with --drop: do not ask for confirmation",
			},
		},
	}
}

func (r *TenantOrphans) Handle(ctx console.Context) error {
	if !tenancy.Enabled() {
		ctx.Warning("tenancy disabled; skip")
		return nil
	}
	orphans, looseAccounts, err := services.FindOrphanTenantStorage()
	if err != nil {
		return err
	}
	if len(orphans) == 0 {
		ctx.Info("no orphaned tenant databases or schemas")
	}
	for _, o := range orphans {
		line := fmt.Sprintf("  %-8s %s (code %s)", o.Kind, o.Name, o.Code)
		if o.Account != "" {
			line += ", account " + o.Account
		}
		ctx.Line(line)
	}
	if len(looseAccounts) > 0 {
		ctx.Warning(fmt.Sprintf("%d t_ account(s) no tenant uses (listed only, never dropped here):", len(looseAccounts)))
		for _, a := range looseAccounts {
			ctx.Line("  account  " + a)
		}
	}
	if len(orphans) == 0 || !ctx.OptionBool("drop") {
		if len(orphans) > 0 {
			ctx.Info(fmt.Sprintf("%d orphan(s); nothing changed. Back up what you may need, then run with --drop.", len(orphans)))
		}
		return nil
	}
	if !ctx.OptionBool("yes") && !ctx.Confirm(fmt.Sprintf("Drop these %d database(s)/schema(s) for good?", len(orphans))) {
		ctx.Info("cancelled")
		return nil
	}
	dropped, failed := 0, 0
	for _, o := range orphans {
		if err := services.DropOrphanTenantStorage(o); err != nil {
			failed++
			ctx.Error(fmt.Sprintf("  %s: %v", o.Name, err))
			continue
		}
		dropped++
		ctx.Line("  dropped " + o.Name)
	}
	ctx.Info(fmt.Sprintf("dropped %d, failed %d", dropped, failed))
	if failed > 0 {
		return fmt.Errorf("%d orphan(s) could not be dropped", failed)
	}
	return nil
}
