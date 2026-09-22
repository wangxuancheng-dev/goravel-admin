package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"
	"github.com/spf13/cast"

	"goravel/app/models"
	"goravel/app/services"
	"goravel/app/tenancy"
)

// TenantScopeFlag returns the shared --tenant option for ops commands.
func TenantScopeFlag() command.Flag {
	return &command.StringFlag{
		Name:    "tenant",
		Aliases: []string{"T"},
		Usage:   "tenant code or id; when tenancy is on and omitted, iterate ready tenants (paged when fleet is large)",
	}
}

// TenantScopePagingFlags are optional --scope-limit / --scope-after-id / --rotate for fleet sweeps.
func TenantScopePagingFlags() []command.Flag {
	return []command.Flag{
		&command.IntFlag{
			Name:  "scope-limit",
			Usage: "max tenants this run (0=auto from TENANCY_SCOPE_*; -1=all)",
		},
		&command.StringFlag{
			Name:  "scope-after-id",
			Usage: "exclusive id cursor (id > after-id); ignored when unlimited",
		},
		&command.BoolFlag{
			Name:  "rotate",
			Usage: "persist id cursor in cache for the next scheduled sweep",
		},
	}
}

// RunTenantScoped executes work on one tenant, all/paged ready tenants, or the default DB when tenancy is off.
func RunTenantScoped(ctx console.Context, work services.TenantWork) error {
	return RunTenantScopedWithOptions(ctx, services.TenantScopeOptions{}, work)
}

// RunTenantScopedRotating is for scheduled jobs: auto-page large fleets and advance a named cursor.
func RunTenantScopedRotating(ctx console.Context, rotateKey string, work services.TenantWork) error {
	return RunTenantScopedWithOptions(ctx, services.TenantScopeOptions{
		Rotate:    true,
		RotateKey: rotateKey,
	}, work)
}

// RunTenantScopedWithOptions merges CLI flags with opts then runs TenantConnectionService.RunTenantScopeWithOptions.
func RunTenantScopedWithOptions(ctx console.Context, opts services.TenantScopeOptions, work services.TenantWork) error {
	svc := services.NewTenantConnectionService()
	hint := strings.TrimSpace(ctx.Option("tenant"))
	if hint == "" {
		hint = strings.TrimSpace(opts.Hint)
	}
	opts.Hint = hint

	if raw := strings.TrimSpace(ctx.Option("scope-limit")); raw != "" {
		opts.Limit = ctx.OptionInt("scope-limit")
		if raw == "-1" {
			opts.Limit = -1
		}
	}
	if raw := strings.TrimSpace(ctx.Option("scope-after-id")); raw != "" {
		opts.AfterID = cast.ToUint(raw)
	}
	if ctx.OptionBool("rotate") {
		opts.Rotate = true
	}

	if tenancy.Enabled() && opts.Hint == "" {
		if opts.Rotate {
			ctx.Info("tenancy on: rotating paged sweep over ready tenants (use --tenant to pin one)")
		} else {
			ctx.Info("tenancy on: iterating ready tenants (paged when fleet is large; use --tenant to pin one)")
		}
	}
	return svc.RunTenantScopeWithOptions(opts, func(tenant *models.Tenant, bound context.Context) error {
		if tenant != nil {
			ctx.Info(fmt.Sprintf("[%s] ...", tenant.Code))
		}
		if err := work(tenant, bound); err != nil {
			if tenant != nil {
				ctx.Error(fmt.Sprintf("[%s] %v", tenant.Code, err))
			}
			return err
		}
		if tenant != nil {
			ctx.Success(fmt.Sprintf("[%s] done", tenant.Code))
		}
		return nil
	})
}

// RunTenantScopedRequire requires --tenant when tenancy is on (for write-heavy commands).
func RunTenantScopedRequire(ctx console.Context, work services.TenantWork) error {
	if tenancy.Enabled() && strings.TrimSpace(ctx.Option("tenant")) == "" {
		return fmt.Errorf("tenancy on: pass --tenant={code|id} (refusing fleet-wide write)")
	}
	return RunTenantScoped(ctx, work)
}
