package services

import (
	"context"

	apperrors "goravel/app/errors"
	"goravel/app/models"

	"goravel-payment"
)

// PaymentGatewayService: third-party create/query/notify (separate from payment CRUD).
// Drivers: goravel-payment/gateways; registry: goravel-payment; DB apply: ApplyPaidResult only.
// Docs: website/docs/advanced/payments.md
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

// CreatePaymentOrder creates a third-party pay order via the registered gateway.
func (s *PaymentGatewayServiceImpl) CreatePaymentOrder(pay *models.Payment, clientIP string) (map[string]any, error) {
	paymentMethod, err := s.paymentMethods.GetPaymentMethodByID(pay.PaymentMethodID)
	if err != nil {
		return nil, err
	}
	config, err := payment.ParseConfigJSON(paymentMethod.Config)
	if err != nil {
		return nil, mapGatewayDriverError(err)
	}
	driver, err := requirePaymentGateway(paymentMethod.Type)
	if err != nil {
		return nil, err
	}
	out, err := driver.Create(s.ctx, paymentOrderFromModel(pay), config, clientIP)
	return out, mapGatewayDriverError(err)
}

// QueryPaymentOrder queries third-party payment status.
func (s *PaymentGatewayServiceImpl) QueryPaymentOrder(pay *models.Payment) (map[string]any, error) {
	paymentMethod, err := s.paymentMethods.GetPaymentMethodByID(pay.PaymentMethodID)
	if err != nil {
		return nil, err
	}
	config, err := payment.ParseConfigJSON(paymentMethod.Config)
	if err != nil {
		return nil, mapGatewayDriverError(err)
	}
	driver, err := requirePaymentGateway(paymentMethod.Type)
	if err != nil {
		return nil, err
	}
	out, err := driver.Query(s.ctx, paymentOrderFromModel(pay), config)
	return out, mapGatewayDriverError(err)
}

// HandlePaymentNotify verifies the callback via the driver, then ApplyPaidResult.
func (s *PaymentGatewayServiceImpl) HandlePaymentNotify(paymentMethod *models.PaymentMethod, notifyData map[string]any) (*models.Payment, error) {
	if paymentMethod == nil {
		return nil, apperrors.ErrInvalidPaymentType
	}
	driver, err := requirePaymentGateway(paymentMethod.Type)
	if err != nil {
		return nil, err
	}
	config, err := payment.ParseConfigJSON(paymentMethod.Config)
	if err != nil {
		return nil, mapGatewayDriverError(err)
	}
	result, err := driver.Notify(s.ctx, config, notifyData)
	if err != nil {
		return nil, mapGatewayDriverError(err)
	}
	if result == nil {
		return nil, apperrors.ErrPaymentNotifyInvalid
	}
	return ApplyPaidResult(s.ctx, *result)
}
