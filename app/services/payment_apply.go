package services

import (
	"context"
	"math"
	"strings"
	"time"

	apperrors "goravel/app/errors"
	"goravel/app/models"
)

// PaidResult is the normalized outcome after a gateway verifies a successful payment.
// Gateways (mock / wechat / alipay) should only verify + map fields; business writes go through ApplyPaidResult.
type PaidResult struct {
	PaymentNo    string
	ThirdPartyNo string
	PayTime      *time.Time
	Amount       *float64 // optional; when set, must match payment.Amount
	NotifyData   map[string]any
}

// ApplyPaidResult marks a payment paid (idempotent) and syncs the related order when still pending.
//
// Sharding: payment is located by payment_no (PAY+YYYYMMDD → payments_YYYYMM);
// order is located by payment.OrderNo (ORD+YYYYMM → orders_YYYYMM). Cross-month
// pay is fine — each no encodes its own create month.
func ApplyPaidResult(ctx context.Context, result PaidResult) (*models.Payment, error) {
	paymentNo := strings.TrimSpace(result.PaymentNo)
	if paymentNo == "" {
		return nil, apperrors.ErrPaymentNoRequired
	}

	payments := NewPaymentService(ctx)
	payment, err := payments.GetPaymentByPaymentNo(paymentNo)
	if err != nil {
		return nil, err
	}

	if payment.Status == models.PaymentStatusPaid {
		return payment, nil
	}
	if payment.Status != models.PaymentStatusPending {
		return nil, apperrors.ErrPaymentStatusInvalid.WithMessage("only pending payments can be marked paid")
	}

	if result.Amount != nil && math.Abs(*result.Amount-payment.Amount) > 0.009 {
		return nil, apperrors.ErrPaymentAmountMismatch
	}

	orders := NewOrderService(ctx)
	order, _, err := orders.GetOrderByOrderNo(payment.OrderNo)
	if err != nil {
		return nil, err
	}
	switch order.Status {
	case models.OrderStatusCancelled:
		return nil, apperrors.ErrOrderNotPayable.WithMessage("order is cancelled")
	case models.OrderStatusPaid, models.OrderStatusPending:
		// ok
	default:
		return nil, apperrors.ErrOrderNotPayable
	}

	payTime := result.PayTime
	if payTime == nil {
		now := time.Now().UTC()
		payTime = &now
	} else {
		utc := payTime.UTC()
		payTime = &utc
	}

	if err := payments.UpdatePaymentStatus(payment.ID, models.PaymentStatusPaid, result.ThirdPartyNo, payTime, "", result.NotifyData, payment.PaymentNo); err != nil {
		return nil, err
	}

	if order.Status == models.OrderStatusPending {
		if err := orders.UpdateOrderByOrderNo(order.OrderNo, models.OrderStatusPaid, order.Remark); err != nil {
			return nil, err
		}
	}

	return payments.GetPaymentByPaymentNo(paymentNo)
}
