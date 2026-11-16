package commands

import (
	"fmt"
	"strings"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"

	"goravel/app/services"
	"goravel/app/tenancy"
)

// TenantEnsureDBUser creates or rotates dedicated DB accounts for existing same-host
// tenants (backfill after enabling TENANCY_DB_DEDICATED_USER).
type TenantEnsureDBUser struct{}

func (r *TenantEnsureDBUser) Signature() string {
	return "tenant:ensure-db-user"
}

func (r *TenantEnsureDBUser) Description() string {
	return "Create or rotate dedicated DB accounts for existing same-host tenants"
}

func (r *TenantEnsureDBUser) Extend() command.Extend {
	return command.Extend{
		Category: "tenant",
		Flags: []command.Flag{
			&command.BoolFlag{
				Name:  "all",
				Usage: "process every non-deleted tenant (ignore {id|code} argument)",
			},
			&command.BoolFlag{
				Name:  "force",
				Usage: "replace operator-supplied same-host usernames with the reserved dedicated account (old account kept)",
			},
			&command.BoolFlag{
				Name:  "rotate",
				Usage: "also rotate passwords for tenants that already use the dedicated account (default: skip them)",
			},
			&command.BoolFlag{
				Name:  "dry-run",
				Usage: "print actions only; do not create users or update rows",
			},
		},
	}
}

func (r *TenantEnsureDBUser) Handle(ctx console.Context) error {
	if !tenancy.Enabled() {
		ctx.Warning("tenancy disabled; skip")
		return nil
	}
	admin := services.NewTenantAdminService()
	opts := services.DedicatedDBUserEnsureOpts{
		ForceReplaceCustom: ctx.OptionBool("force"),
		RotatePassword:     ctx.OptionBool("rotate"),
		DryRun:             ctx.OptionBool("dry-run"),
	}

	var ids []uint
	if ctx.OptionBool("all") {
		list, err := admin.ListAll()
		if err != nil {
			return err
		}
		for i := range list {
			ids = append(ids, list[i].ID)
		}
	} else {
		idOrCode := strings.TrimSpace(ctx.Argument(0))
		if idOrCode == "" {
			ctx.Error("usage: tenant:ensure-db-user {id|code} | --all [--force] [--rotate] [--dry-run]")
			return nil
		}
		tenant, err := services.NewTenantConnectionService().FindTenantByIDOrCode(idOrCode)
		if err != nil {
			return err
		}
		ids = append(ids, tenant.ID)
	}

	created, rotated, skipped, failed := 0, 0, 0, 0
	for _, id := range ids {
		res, err := admin.EnsureDedicatedDBUser(id, opts)
		line := fmt.Sprintf("  #%d %s: %s - %s", res.TenantID, res.Code, res.Action, res.Detail)
		if err != nil {
			failed++
			ctx.Error(line + fmt.Sprintf(" (%v)", err))
			continue
		}
		switch res.Action {
		case services.DedicatedDBUserActionCreated:
			created++
			ctx.Info(line)
		case services.DedicatedDBUserActionRotated:
			rotated++
			ctx.Info(line)
		default:
			skipped++
			ctx.Line(line)
		}
	}
	prefix := ""
	if opts.DryRun {
		prefix = "dry-run: "
	}
	ctx.Success(fmt.Sprintf("%screated %d, rotated %d, skipped %d, failed %d", prefix, created, rotated, skipped, failed))
	if failed > 0 {
		return fmt.Errorf("%d tenant(s) failed", failed)
	}
	return nil
}
