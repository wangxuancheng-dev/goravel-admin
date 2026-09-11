package support

import (
	"github.com/goravel/framework/contracts/event"
	"github.com/goravel/framework/facades"

	"goravel/app/events"
	"goravel/app/search"
)

// RequestOrderSearchSync 通过事件异步入队搜索同步（解耦 Service 与队列细节）。
func RequestOrderSearchSync(orderID uint, orderNo string, op string, tenantID uint) {
	if !search.OrdersSyncEnabled() {
		return
	}
	args := events.OrderSearchSyncArgs{
		OrderID:  orderID,
		OrderNo:  orderNo,
		Op:       op,
		TenantID: tenantID,
	}
	if err := facades.Event().Job(&events.OrderSearchSync{}, []event.Arg{
		{Type: "any", Value: args},
	}).Dispatch(); err != nil {
		facades.Log().Errorf("order search sync event dispatch failed: order_id=%d op=%s err=%v", orderID, op, err)
	}
}
