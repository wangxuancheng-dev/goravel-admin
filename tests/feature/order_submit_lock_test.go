package feature_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/goravel/framework/facades"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"goravel/app/services"
	"goravel/app/tenancy"
)

// A second submit with the same request_id while the first still holds the lock must be
// rejected before anything is written. (The old check read the lock with
// Cache.Get(key, &v), which never fills v, so duplicates always went through.)
func TestCreateOrderRejectsInFlightRequestID(t *testing.T) {
	ctx := t.Context()
	requestID := fmt.Sprintf("req-lock-%d", time.Now().UnixNano())
	lockKey := tenancy.CacheKey(ctx, "order:lock:"+requestID)
	require.True(t, facades.Cache().Add(lockKey, "held-by-first-submit", time.Minute))
	t.Cleanup(func() { facades.Cache().Forget(lockKey) })

	order, _, err := services.NewOrderService(ctx).CreateOrder(1, 1, []services.OrderProduct{
		{ProductID: 1, ProductName: "sku", Price: 1, Quantity: 1},
	}, requestID, "duplicate submit")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "订单正在处理中")
	assert.Nil(t, order)
	assert.Equal(t, "held-by-first-submit", facades.Cache().GetString(lockKey), "the loser must not release the winner's lock")
}
