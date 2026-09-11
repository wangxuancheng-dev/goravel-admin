package events

import "github.com/goravel/framework/contracts/event"

// OrderSearchSyncArgs 订单搜索同步事件参数。
type OrderSearchSyncArgs struct {
	OrderID  uint
	OrderNo  string
	Op       string // index | delete
	TenantID uint   // tenancy 开启时传入，供 outbox/队列绑定租户库
}

// OrderSearchSync 订单写入/删除搜索引擎的异步同步事件。
type OrderSearchSync struct{}

func (receiver *OrderSearchSync) Handle(args []event.Arg) ([]event.Arg, error) {
	return args, nil
}
