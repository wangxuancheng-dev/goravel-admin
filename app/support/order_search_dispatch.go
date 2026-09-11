package support

import (
	"encoding/json"

	"github.com/goravel/framework/contracts/event"
	"github.com/goravel/framework/contracts/queue"
	"github.com/goravel/framework/facades"

	"goravel/app/events"
	"goravel/app/queuejobs"
	"goravel/app/search"
)

// RequestOrderSearchSync 通过事件异步入队搜索同步（解耦 Service 与队列细节）。
func RequestOrderSearchSync(orderID uint, orderNo string, op string) {
	if !search.OrdersSyncEnabled() {
		return
	}
	args := events.OrderSearchSyncArgs{
		OrderID: orderID,
		OrderNo: orderNo,
		Op:      op,
	}
	if err := facades.Event().Job(&events.OrderSearchSync{}, []event.Arg{
		{Type: "any", Value: args},
	}).Dispatch(); err != nil {
		facades.Log().Errorf("order search sync event dispatch failed: order_id=%d op=%s err=%v", orderID, op, err)
	}
}

// DispatchOrderSearchSync 写 outbox 并入队同步任务（由监听器调用）。
func DispatchOrderSearchSync(orderID uint, orderNo string, op string) {
	if !search.OrdersSyncEnabled() {
		return
	}
	payload := map[string]any{
		"order_id": orderID,
		"op":       op,
	}
	if orderNo != "" {
		payload["order_no"] = orderNo
	}
	search.CreateSyncOutbox(orderID, orderNo, op, payload)
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		facades.Log().Errorf("order search sync marshal failed: %v", err)
		return
	}
	qargs := []queue.Arg{{Type: "string", Value: string(jsonBytes)}}
	if err := facades.Queue().Job(&queuejobs.SyncOrderSearch{}, qargs).OnQueue(search.SyncQueue()).Dispatch(); err != nil {
		facades.Log().Errorf("order search sync dispatch failed: order_id=%d op=%s err=%v", orderID, op, err)
	}
}
