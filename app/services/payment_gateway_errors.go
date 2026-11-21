package services

import (
	"errors"
	"fmt"
	"strings"

	apperrors "goravel/app/errors"

	"goravel-payment"
)

// gatewaySentinelMap pairs goravel-payment sentinels with host BusinessErrors.
var gatewaySentinelMap = []struct {
	sentinel error
	business *apperrors.BusinessError
}{
	{payment.ErrNotImplemented, apperrors.ErrPaymentGatewayNotImplemented},
	{payment.ErrConfigRequired, apperrors.ErrPaymentConfigRequired},
	{payment.ErrNotifyInvalid, apperrors.ErrPaymentNotifyInvalid},
	{payment.ErrCreateFailed, apperrors.ErrCreatePaymentFailed},
	{payment.ErrStatusInvalid, apperrors.ErrPaymentStatusInvalid},
}

// mapGatewayDriverError maps reusable package sentinels to host BusinessError.
// The message keeps only the driver's detail ("out_trade_no is required"), not the
// package's "payment: notify invalid: " prefix; a bare sentinel keeps the host default.
func mapGatewayDriverError(err error) error {
	if err == nil {
		return nil
	}
	if be, ok := apperrors.GetBusinessError(err); ok {
		return be
	}
	for _, m := range gatewaySentinelMap {
		if !errors.Is(err, m.sentinel) {
			continue
		}
		if detail := gatewayErrorDetail(err, m.sentinel); detail != "" {
			return m.business.WithMessage(detail)
		}
		return m.business
	}
	return err
}

// gatewayErrorDetail strips the sentinel's text from fmt.Errorf("%w: detail", sentinel).
func gatewayErrorDetail(err, sentinel error) string {
	msg := strings.TrimSpace(err.Error())
	if detail, ok := strings.CutPrefix(msg, sentinel.Error()+":"); ok {
		return strings.TrimSpace(detail)
	}
	if msg == sentinel.Error() {
		return ""
	}
	return msg
}

func mapGatewayStatusGateError(err error) error {
	if err == nil {
		return nil
	}
	mapped := mapGatewayDriverError(err)
	if mapped == err {
		return fmt.Errorf("payment status gate: %w", err)
	}
	return mapped
}
