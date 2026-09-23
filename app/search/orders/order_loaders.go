package orders

import (
	"context"

	"goravel/app/models"
)

// Order loaders are app-injected callbacks (not imports of app/services).
// Wired in providers.SearchServiceProvider to avoid cycles
// (services -> search/orders -> services) and to keep this package extractable.
var (
	findOrderByID        func(ctx context.Context, orderID uint, orderNo ...string) (*models.Order, error)
	findOrderWithDetails func(ctx context.Context, orderID uint, orderNoHint string) (*models.Order, []models.OrderDetail, error)
)

// SetOrderLoaders registers sharded order read helpers used by Push.
// Call once from the app provider; do not set from services package init.
func SetOrderLoaders(
	byID func(ctx context.Context, orderID uint, orderNo ...string) (*models.Order, error),
	withDetails func(ctx context.Context, orderID uint, orderNoHint string) (*models.Order, []models.OrderDetail, error),
) {
	findOrderByID = byID
	findOrderWithDetails = withDetails
}
