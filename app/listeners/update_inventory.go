package listeners

import (
	"math"

	"github.com/goravel/framework/contracts/event"
	"github.com/goravel/framework/facades"
	"github.com/spf13/cast"

	"goravel/app/errors"
)

// OrderCreatedArgs 订单创建事件的参数结构体
type OrderCreatedArgs struct {
	OrderID uint
	Remark  string
}

// UpdateInventory 更新库存监听器（同步执行）
type UpdateInventory struct {
}

func (receiver *UpdateInventory) Signature() string {
	return "update_inventory"
}

// Queue 禁用队列，同步执行（库存需要立即更新）
func (receiver *UpdateInventory) Queue(args ...any) event.Queue {
	return event.Queue{
		Enable:     false, // 同步执行，库存需要立即更新
		Connection: "",
		Queue:      "",
	}
}

// Handle 处理更新库存
//
// 参数:
//   - args[0]: OrderCreatedArgs 结构体或 orderID (int/uint)
//
// 返回:
//   - error: 错误信息
func (receiver *UpdateInventory) Handle(args ...any) error {
	if len(args) < 1 {
		return errors.ErrInvalidArgument.WithMessage("missing order ID")
	}

	var orderID uint
	if oca, ok := args[0].(OrderCreatedArgs); ok {
		orderID = oca.OrderID
	} else {
		// 兼容旧版本：按位置解析
		orderID = cast.ToUint(args[0])
	}

	if orderID == 0 {
		return errors.ErrInvalidArgument.WithMessage("invalid order ID")
	}

	// 验证订单ID范围
	if orderID > math.MaxUint32 {
		return errors.ErrInvalidArgument.WithMessage("order ID exceeds maximum value")
	}

	// 二次开发扩展点：在此扣减/锁定 SKU 库存；当前仅记日志便于演示事件链
	facades.Log().Infof("[同步] 库存更新 hook（未接库存域时仅记日志），订单 ID: %d", orderID)
	return nil
}
