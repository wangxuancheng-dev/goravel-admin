package listeners

import (
	"github.com/goravel/framework/contracts/event"

	"goravel/app/errors"
	"goravel/app/events"
	"goravel/app/search"
	"goravel/app/support"
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
			OrderID: uintFromAny(v["order_id"]),
			OrderNo: stringFromAny(v["order_no"]),
			Op:      stringFromAny(v["op"]),
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

	support.DispatchOrderSearchSync(syncArgs.OrderID, syncArgs.OrderNo, syncArgs.Op)
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
