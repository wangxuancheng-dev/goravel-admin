package orders

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"goravel/app/models"
)

func TestParseListTimeRangeDefaults(t *testing.T) {
	start, end, err := ParseListTimeRange("", "")
	require.NoError(t, err)
	assert.True(t, end.IsZero())
	ago := time.Now().UTC().AddDate(0, 0, -7)
	if start.Before(ago.Add(-2*time.Minute)) || start.After(ago.Add(2*time.Minute)) {
		t.Fatalf("default start should be ~7 days ago, got %v", start)
	}
}

func TestParseListTimeRangeInvalid(t *testing.T) {
	_, _, err := ParseListTimeRange("bad-time", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid_start_time")

	_, _, err = ParseListTimeRange("2026-01-01 00:00:00", "bad-end")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid_end_time")
}

func TestParseListTimeRangeValid(t *testing.T) {
	start, end, err := ParseListTimeRange("2026-01-01 00:00:00", "2026-01-31 23:59:59")
	require.NoError(t, err)
	assert.Equal(t, 2026, start.Year())
	assert.Equal(t, time.January, start.Month())
	assert.Equal(t, 31, end.Day())
}

func TestDetailsTableFromOrdersTable(t *testing.T) {
	assert.Equal(t, "order_details_202501", DetailsTableFromOrdersTable("orders_202501"))
	assert.Equal(t, "orders", DetailsTableFromOrdersTable("orders"))
	assert.Equal(t, "foo_order_details_bar", DetailsTableFromOrdersTable("foo_orders_bar"))
}

func TestToJSONAndWithDetails(t *testing.T) {
	order := models.Order{
		OrderNo: "ORD202601",
		UserID:  9,
		Amount:  12.5,
		Status:  "pending",
		Remark:  "note",
	}
	order.ID = 1

	payload := ToJSON(order)
	assert.Equal(t, uint(1), payload["id"])
	assert.Equal(t, "ORD202601", payload["order_no"])
	assert.Equal(t, uint(9), payload["user_id"])
	assert.Equal(t, 12.5, payload["amount"])
	assert.Equal(t, "pending", payload["status"])

	detail := models.OrderDetail{
		OrderID:     1,
		ProductID:   2,
		ProductName: "sku",
		Price:       6.25,
		Quantity:    2,
		Subtotal:    12.5,
	}
	detail.ID = 10

	item := &WithDetails{Order: order, Details: []models.OrderDetail{detail}}
	full := WithDetailsToJSON(item)
	require.NotNil(t, full)
	details, ok := full["details"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, details, 1)
	assert.Equal(t, uint(10), details[0]["id"])
	assert.Equal(t, "sku", details[0]["product_name"])

	assert.Nil(t, WithDetailsToJSON(nil))
}

func TestDetailsTablePrefixOnlyOnce(t *testing.T) {
	got := DetailsTableFromOrdersTable("orders_202501")
	assert.True(t, strings.HasPrefix(got, "order_details_"))
	assert.False(t, strings.Contains(got, "orders_"))
}
