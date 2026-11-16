package feature_test

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/app/tenancyctx"
)

// createOrderWithCollision makes the first generated order number collide with an
// existing order: CreateOrder must recognise the duplicate key, roll back to its
// savepoint and retry with the next number inside the same transaction (a failed
// INSERT aborts a PostgreSQL transaction).
func createOrderWithCollision(t *testing.T, ctx context.Context, label string) {
	t.Helper()
	svc := services.NewOrderService(ctx)
	products := []services.OrderProduct{{ProductID: 1, ProductName: "sku", Price: 1, Quantity: 1}}

	first, _, err := svc.CreateOrder(1, 1, products, "req-collide-a-"+label, "collide a")
	require.NoError(t, err, "%s: first order", label)

	prev := services.GenerateOrderNoFn
	t.Cleanup(func() { services.GenerateOrderNoFn = prev })
	var calls atomic.Int32
	services.GenerateOrderNoFn = func() string {
		if calls.Add(1) == 1 {
			return first.OrderNo // taken
		}
		return prev()
	}

	second, _, err := svc.CreateOrder(1, 1, products, "req-collide-b-"+label, "collide b")
	require.NoError(t, err, "%s: an order number collision must be retried", label)
	assert.NotEqual(t, first.OrderNo, second.OrderNo)
	assert.GreaterOrEqual(t, calls.Load(), int32(2), "%s: a second number was generated", label)

	got, _, err := services.NewOrderService(ctx).GetOrderByOrderNo(second.OrderNo)
	require.NoError(t, err, "%s: the retried order was committed", label)
	assert.Equal(t, "collide b", got.Remark)
}

// TestOrmQueryUnavailableConnection: a ctx bound to a tenant connection that cannot be
// built gets a query whose calls fail (no nil-pointer panic in a chain, and no
// fallback to the platform database), while system logs still reach the platform.
func TestOrmQueryUnavailableConnection(t *testing.T) {
	withTenancyDriver(t, "database")
	ctx := tenancyctx.WithTenant(context.Background(), 987654321, "tenant_987654321", "nosuchtenant")

	q := appfacades.OrmQuery(ctx)
	require.NotNil(t, q)
	down, reason := appfacades.QueryUnavailable(q)
	require.True(t, down, "the connection of a missing tenant cannot be built")
	require.Error(t, reason)

	var admin models.Admin
	err := appfacades.OrmQuery(ctx).Where("id", 1).Where("status", 1).OrderBy("id").First(&admin)
	require.Error(t, err, "a chained query on an unavailable connection fails")
	assert.Zero(t, admin.ID, "nothing was read from another database")
	n, err := appfacades.OrmQuery(ctx).Model(&models.Admin{}).Count()
	assert.Error(t, err)
	assert.Zero(t, n)
	assert.Equal(t, "", appfacades.OrmDriver(ctx))

	logQ := appfacades.SystemLogOrmQuery(ctx)
	down, _ = appfacades.QueryUnavailable(logQ)
	assert.False(t, down, "system logs of a broken tenant connection go to the platform database")
}

// TestOrderNoCollisionOnPlatform runs the retry on the platform database (MySQL in CI,
// or PostgreSQL with DB_CONNECTION=postgres).
func TestOrderNoCollisionOnPlatform(t *testing.T) {
	withTenancyDriver(t, "off")
	createOrderWithCollision(t, t.Context(), "platform")
}
