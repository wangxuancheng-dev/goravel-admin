package listeners

import (
	"math"

	"github.com/goravel/framework/contracts/event"
	"github.com/goravel/framework/facades"
	"github.com/spf13/cast"

	"goravel/app/errors"
)

// OrderShippedArgs 订单发货事件的参数结构体
type OrderShippedArgs struct {
	OrderID uint
	Remark  string
}

type SendShipmentNotification struct {
}

func NewSendShipmentNotification() *SendShipmentNotification {
	return &SendShipmentNotification{}
}

func (receiver *SendShipmentNotification) Signature() string {
	return "send_shipment_notification"
}

func (receiver *SendShipmentNotification) Queue(args ...any) event.Queue {
	return event.Queue{
		Enable:     true,
		Connection: "",
		Queue:      "",
	}
}

// Handle 处理发送发货通知
//
// 参数:
//   - args[0]: OrderShippedArgs 结构体或 orderID/remark (int/uint/string)
//
// 返回:
//   - error: 错误信息
func (receiver *SendShipmentNotification) Handle(args ...any) error {
	if len(args) < 1 {
		return errors.ErrInvalidArgument.WithMessage("missing shipment notification arguments")
	}

	var orderID uint
	var remark string

	if osa, ok := args[0].(OrderShippedArgs); ok {
		orderID = osa.OrderID
		remark = osa.Remark
	} else {
		// 兼容旧版本：按位置解析
		orderID = cast.ToUint(args[0])
		if len(args) > 1 {
			remark = cast.ToString(args[1])
		}
	}

	if orderID == 0 {
		return errors.ErrInvalidArgument.WithMessage("invalid order ID")
	}

	// 验证订单ID范围
	if orderID > math.MaxUint32 {
		return errors.ErrInvalidArgument.WithMessage("order ID exceeds maximum value")
	}

	// 限制备注长度，防止过长字符串
	const maxRemarkLength = 1000
	if len(remark) > maxRemarkLength {
		remark = remark[:maxRemarkLength]
		facades.Log().Infof("[队列] 发货通知备注过长，已截断至 %d 字符", maxRemarkLength)
	}

	facades.Log().Infof("[队列] 发货通知 hook（未接渠道时仅记日志），订单 ID: %d, 备注: %s", orderID, remark)
	return nil
}
