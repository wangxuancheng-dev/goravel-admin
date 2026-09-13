package services

import (
	"context"

	apperrors "goravel/app/errors"
	"goravel/app/models"
	apppayment "goravel/app/payment"
)

// PaymentGatewayService 第三方支付下单/查询/回调（与后台支付记录 CRUD 解耦）。
// 渠道实现在 app/payment/gateways；注册表在 app/payment；落库统一 ApplyPaidResult。
// 文档：website/docs/advanced/payments.md
type PaymentGatewayService interface {
	CreatePaymentOrder(payment *models.Payment, clientIP string) (map[string]any, error)
	QueryPaymentOrder(payment *models.Payment) (map[string]any, error)
	HandlePaymentNotify(paymentMethod *models.PaymentMethod, notifyData map[string]any) (*models.Payment, error)
}

type PaymentGatewayServiceImpl struct {
	ctx            context.Context
	paymentMethods PaymentMethodService
}

func NewPaymentGatewayService(ctx context.Context) PaymentGatewayService {
	return &PaymentGatewayServiceImpl{
		ctx:            ctx,
		paymentMethods: NewPaymentMethodService(ctx),
	}
}

// CreatePaymentOrder 创建支付订单（调用已注册网关）
func (s *PaymentGatewayServiceImpl) CreatePaymentOrder(payment *models.Payment, clientIP string) (map[string]any, error) {
	paymentMethod, err := s.paymentMethods.GetPaymentMethodByID(payment.PaymentMethodID)
	if err != nil {
		return nil, err
	}
	config, err := apppayment.ParseMethodConfig(paymentMethod)
	if err != nil {
		return nil, err
	}
	driver, err := requirePaymentGateway(paymentMethod.Type)
	if err != nil {
		return nil, err
	}
	return driver.Create(s.ctx, payment, paymentMethod, config, clientIP)
}

// QueryPaymentOrder 查询支付订单状态
func (s *PaymentGatewayServiceImpl) QueryPaymentOrder(payment *models.Payment) (map[string]any, error) {
	paymentMethod, err := s.paymentMethods.GetPaymentMethodByID(payment.PaymentMethodID)
	if err != nil {
		return nil, err
	}
	config, err := apppayment.ParseMethodConfig(paymentMethod)
	if err != nil {
		return nil, err
	}
	driver, err := requirePaymentGateway(paymentMethod.Type)
	if err != nil {
		return nil, err
	}
	return driver.Query(s.ctx, payment, paymentMethod, config)
}

// HandlePaymentNotify 处理支付回调：驱动验签映射 PaidResult，本层 ApplyPaidResult 落库。
func (s *PaymentGatewayServiceImpl) HandlePaymentNotify(paymentMethod *models.PaymentMethod, notifyData map[string]any) (*models.Payment, error) {
	if paymentMethod == nil {
		return nil, apperrors.ErrInvalidPaymentType
	}
	driver, err := requirePaymentGateway(paymentMethod.Type)
	if err != nil {
		return nil, err
	}
	result, err := driver.Notify(s.ctx, paymentMethod, notifyData)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, apperrors.ErrPaymentNotifyInvalid
	}
	return ApplyPaidResult(s.ctx, *result)
}
