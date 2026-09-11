package queuejobs

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/goravel/framework/facades"
	"github.com/spf13/cast"

	"goravel/app/errors"
	"goravel/app/search"
	searchorders "goravel/app/search/orders"
	"goravel/app/utils"
)

// SyncOrderSearch 订单搜索同步队列任务。
type SyncOrderSearch struct{}

func (r *SyncOrderSearch) Signature() string {
	return "sync_order_search"
}

func (r *SyncOrderSearch) Handle(args ...any) error {
	if !search.OrdersSyncEnabled() {
		return nil
	}
	if len(args) < 1 {
		return errors.ErrInvalidArgument.WithMessage("missing sync args")
	}
	var m map[string]any
	switch v := args[0].(type) {
	case map[string]any:
		m = v
	case string:
		if err := json.Unmarshal([]byte(v), &m); err != nil {
			return errors.ErrInvalidArgument.WithMessage("sync args json invalid")
		}
	default:
		return errors.ErrInvalidArgument.WithMessage("sync args must be map or json string")
	}
	if m == nil {
		return errors.ErrInvalidArgument.WithMessage("sync args empty")
	}
	orderID := cast.ToUint(m["order_id"])
	if orderID == 0 {
		return errors.ErrInvalidArgument.WithMessage("invalid order_id")
	}
	op, _ := utils.GetString(m, "op")
	if op != "index" && op != "delete" {
		return fmt.Errorf("invalid op: %s", op)
	}
	orderNo, _ := utils.GetString(m, "order_no")
	ctx := context.Background()
	if err := searchorders.Push(ctx, orderID, orderNo, op); err != nil {
		facades.Log().Errorf("SyncOrderSearch failed: order_id=%d op=%s err=%v", orderID, op, err)
		search.MarkSyncOutboxFailed(orderID, op, err.Error())
		return err
	}
	search.MarkSyncOutboxProcessed(orderID, op)
	return nil
}
