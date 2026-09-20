package commands

import (
	"context"
	"fmt"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"

	"goravel/app/models"
	"goravel/app/services"
)

type CancelExpiredOrders struct{}

func (r *CancelExpiredOrders) Signature() string {
	return "order:cancel-expired"
}

func (r *CancelExpiredOrders) Description() string {
	return "Demo: cancel pending orders past expire_at (Delay job safety net)"
}

func (r *CancelExpiredOrders) Extend() command.Extend {
	return command.Extend{
		Category: "order",
		Flags:    []command.Flag{TenantScopeFlag()},
	}
}

func (r *CancelExpiredOrders) Handle(ctx console.Context) error {
	return RunTenantScoped(ctx, func(_ *models.Tenant, bound context.Context) error {
		n, err := services.CancelExpiredOrdersSweep(bound)
		if err != nil {
			return fmt.Errorf("order:cancel-expired failed: %w", err)
		}
		ctx.Info(fmt.Sprintf("expired pending orders cancelled=%d", n))
		return nil
	})
}
