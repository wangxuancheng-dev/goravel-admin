package events

import "github.com/goravel/framework/contracts/event"

// OrderSearchSyncArgs 订单搜索同步事件参数。
type OrderSearchSyncArgs struct {
	OrderID uint
	OrderNo string
	Op      string // index | delete
}

// OrderSearchSync 订单写入/删除搜索引擎的异步同步事件。
type OrderSearchSync struct{}

func (receiver *OrderSearchSync) Handle(args []event.Arg) ([]event.Arg, error) {
	return args, nil
}
