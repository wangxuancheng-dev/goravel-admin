package orders

import (
	"context"

	"goravel/app/models"
)

// Order loaders are wired from app/services in SearchServiceProvider to avoid
// an import cycle (services -> search/orders -> services).
var (
	findOrderByID        func(ctx context.Context, orderID uint, orderNo ...string) (*models.Order, error)
	findOrderWithDetails func(ctx context.Context, orderID uint, orderNoHint string) (*models.Order, []models.OrderDetail, error)
)

// SetOrderLoaders registers sharded order read helpers used by Push.
func SetOrderLoaders(
	byID func(ctx context.Context, orderID uint, orderNo ...string) (*models.Order, error),
	withDetails func(ctx context.Context, orderID uint, orderNoHint string) (*models.Order, []models.OrderDetail, error),
) {
	findOrderByID = byID
	findOrderWithDetails = withDetails
}
