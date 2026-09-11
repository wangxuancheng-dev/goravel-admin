package listeners

import (
	"context"
	"encoding/json"

	"github.com/goravel/framework/contracts/event"
	"github.com/goravel/framework/contracts/queue"
	"github.com/goravel/framework/facades"

	"goravel/app/errors"
	"goravel/app/events"
	"goravel/app/queuejobs"
	"goravel/app/search"
	"goravel/app/services"
)

// SyncOrderSearch 订单搜索同步监听器（异步入队）。
type SyncOrderSearch struct{}

func (receiver *SyncOrderSearch) Signature() string {
	return "sync_order_search_listener"
}

func (receiver *SyncOrderSearch) Queue(args ...any) event.Queue {
	return event.Queue{
		Enable:     true,
		Connection: "",
		Queue:      search.SyncQueue(),
	}
}

func (receiver *SyncOrderSearch) Handle(args ...any) error {
	if len(args) < 1 {
		return errors.ErrInvalidArgument.WithMessage("missing order search sync args")
	}

	var syncArgs events.OrderSearchSyncArgs
	switch v := args[0].(type) {
	case events.OrderSearchSyncArgs:
		syncArgs = v
	case map[string]any:
		syncArgs = events.OrderSearchSyncArgs{
			OrderID:  uintFromAny(v["order_id"]),
			OrderNo:  stringFromAny(v["order_no"]),
			Op:       stringFromAny(v["op"]),
			TenantID: uintFromAny(v["tenant_id"]),
		}
	default:
		return errors.ErrInvalidArgument.WithMessage("invalid order search sync args")
	}

	if syncArgs.OrderID == 0 {
		return errors.ErrInvalidArgument.WithMessage("invalid order_id")
	}
	if syncArgs.Op != "index" && syncArgs.Op != "delete" {
		return errors.ErrInvalidArgument.WithMessage("invalid op")
	}
	if !search.OrdersSyncEnabled() {
		return nil
	}

	ctx := context.Background()
	if syncArgs.TenantID > 0 {
		bound, err := services.NewTenantConnectionService().BindBackground(ctx, syncArgs.TenantID)
		if err != nil {
			facades.Log().Errorf("order search sync bind tenant failed: tenant_id=%d err=%v", syncArgs.TenantID, err)
			return err
		}
		ctx = bound
	}

	payload := map[string]any{
		"order_id": syncArgs.OrderID,
		"op":       syncArgs.Op,
	}
	if syncArgs.OrderNo != "" {
		payload["order_no"] = syncArgs.OrderNo
	}
	if syncArgs.TenantID > 0 {
		payload["tenant_id"] = syncArgs.TenantID
	}
	search.CreateSyncOutbox(ctx, syncArgs.OrderID, syncArgs.OrderNo, syncArgs.Op, payload)

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		facades.Log().Errorf("order search sync marshal failed: %v", err)
		return nil
	}
	qargs := []queue.Arg{{Type: "string", Value: string(jsonBytes)}}
	if err := facades.Queue().Job(&queuejobs.SyncOrderSearch{}, qargs).OnQueue(search.SyncQueue()).Dispatch(); err != nil {
		facades.Log().Errorf("order search sync dispatch failed: order_id=%d op=%s err=%v", syncArgs.OrderID, syncArgs.Op, err)
	}
	return nil
}

func uintFromAny(v any) uint {
	switch n := v.(type) {
	case uint:
		return n
	case int:
		if n > 0 {
			return uint(n)
		}
	case int64:
		if n > 0 {
			return uint(n)
		}
	case float64:
		if n > 0 {
			return uint(n)
		}
	}
	return 0
}

func stringFromAny(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
