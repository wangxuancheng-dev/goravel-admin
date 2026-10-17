package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"

	"goravel/app/models"
	"goravel/app/services"
)

type FillLoginLocations struct{}

func (r *FillLoginLocations) Signature() string {
	return "login_log:fill-location"
}

func (r *FillLoginLocations) Description() string {
	return "Fill empty login-log locations from ip-api.com (same IP is queried once)"
}

func (r *FillLoginLocations) Extend() command.Extend {
	return command.Extend{
		Category: "login_log",
		Flags: []command.Flag{
			TenantScopeFlag(),
			&command.BoolFlag{
				Name:  "platform",
				Usage: "Fill platform_login_logs instead of tenant login_logs",
			},
			&command.IntFlag{
				Name:    "limit",
				Aliases: []string{"l"},
				Value:   200,
				Usage:   "Max distinct IPs to look up in this run",
			},
		},
	}
}

func (r *FillLoginLocations) Handle(ctx console.Context) error {
	limit := ctx.OptionInt("limit")
	if limit <= 0 {
		limit = 200
	}
	pause := 1400 * time.Millisecond
	if ctx.OptionBool("platform") {
		ctx.Info(fmt.Sprintf("Filling empty platform login locations, limit=%d", limit))
		result, err := services.FillPlatformEmptyLoginLocations(limit, pause)
		if err != nil {
			return fmt.Errorf("fill platform login locations failed: %w", err)
		}
		ctx.Info(fmt.Sprintf("updated=%d looked_up=%d failed=%d has_more=%v", result.Updated, result.LookedUp, result.Failed, result.HasMore))
		return nil
	}
	return RunTenantScoped(ctx, func(_ *models.Tenant, bound context.Context) error {
		ctx.Info(fmt.Sprintf("Filling empty login locations, limit=%d", limit))
		result, err := services.NewLoginLogService(bound).FillEmptyLocationsBatch(limit, pause)
		if err != nil {
			return fmt.Errorf("fill login locations failed: %w", err)
		}
		ctx.Info(fmt.Sprintf("updated=%d looked_up=%d failed=%d has_more=%v", result.Updated, result.LookedUp, result.Failed, result.HasMore))
		return nil
	})
}
