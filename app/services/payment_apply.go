package services

import (
	"context"
	"math"
	"strings"
	"time"

	apperrors "goravel/app/errors"
	"goravel/app/models"
	apppayment "goravel/app/payment"
)

// PaidResult re-exports the domain type for backward-compatible imports in services/gateways/tests.
type PaidResult = apppayment.PaidResult

// ApplyPaidResult marks a payment paid (idempotent) and syncs the related order when still pending.
//
// Sharding: payment is located by payment_no (PAY+YYYYMMDD → payments_YYYYMM);
// order is located by payment.OrderNo (ORD+YYYYMM → orders_YYYYMM). Cross-month
// pay is fine — each no encodes its own create month.
// Status gate lives in goravel/app/payment (ApplyPaidResultStatusGate).
func ApplyPaidResult(ctx context.Context, result PaidResult) (*models.Payment, error) {
	paymentNo := strings.TrimSpace(result.PaymentNo)
	if paymentNo == "" {
		return nil, apperrors.ErrPaymentNoRequired
	}

	payments := NewPaymentService(ctx)
	pay, err := payments.GetPaymentByPaymentNo(paymentNo)
	if err != nil {
		return nil, err
	}

	skip, err := apppayment.ApplyPaidResultStatusGate(pay.Status)
	if err != nil {
		return nil, err
	}
	if skip {
		return pay, nil
	}

	if result.Amount != nil && math.Abs(*result.Amount-pay.Amount) > 0.009 {
		return nil, apperrors.ErrPaymentAmountMismatch
	}

	orders := NewOrderService(ctx)
	order, _, err := orders.GetOrderByOrderNo(pay.OrderNo)
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

	if err := payments.UpdatePaymentStatus(pay.ID, models.PaymentStatusPaid, result.ThirdPartyNo, payTime, "", result.NotifyData, pay.PaymentNo); err != nil {
		return nil, err
	}

	if order.Status == models.OrderStatusPending {
		if err := orders.UpdateOrderByOrderNo(order.OrderNo, models.OrderStatusPaid, order.Remark); err != nil {
			// Best-effort compensate: payment already marked paid on a (possibly different) shard table.
			_ = payments.UpdatePaymentStatus(pay.ID, models.PaymentStatusPending, "", nil, "order sync failed: "+err.Error(), nil, pay.PaymentNo)
			return nil, err
		}
	}

	return payments.GetPaymentByPaymentNo(paymentNo)
}
