package listeners

import (
	"math"

	"github.com/goravel/framework/contracts/event"
	"github.com/goravel/framework/facades"
	"github.com/spf13/cast"

	"goravel/app/errors"
)

// SendOrderNotification 发送订单通知监听器（启用队列）
type SendOrderNotification struct {
}

func (receiver *SendOrderNotification) Signature() string {
	return "send_order_notification"
}

// Queue 启用队列，异步发送通知
func (receiver *SendOrderNotification) Queue(args ...any) event.Queue {
	return event.Queue{
		Enable:     true,
		Connection: "",
		Queue:      "notifications", // 使用专门的通知队列
	}
}

// Handle 处理发送订单通知
//
// 参数:
//   - args[0]: OrderCreatedArgs 结构体或 orderID (int/uint)
//
// 返回:
//   - error: 错误信息
func (receiver *SendOrderNotification) Handle(args ...any) error {
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

	// 二次开发扩展点：在此接入短信 / 站内信 / 推送等真实通知渠道
	facades.Log().Infof("[队列] 订单通知 hook（未接渠道时仅记日志），订单 ID: %d", orderID)
	return nil
}
