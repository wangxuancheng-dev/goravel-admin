package services

import (
	"errors"
	"fmt"

	apperrors "goravel/app/errors"

	"goravel-payment"
)

// mapGatewayDriverError maps reusable package sentinels to host BusinessError.
func mapGatewayDriverError(err error) error {
	if err == nil {
		return nil
	}
	if be, ok := apperrors.GetBusinessError(err); ok {
		return be
	}
	switch {
	case errors.Is(err, payment.ErrNotImplemented):
		return apperrors.ErrPaymentGatewayNotImplemented
	case errors.Is(err, payment.ErrConfigRequired):
		return apperrors.ErrPaymentConfigRequired.WithMessage(err.Error())
	case errors.Is(err, payment.ErrNotifyInvalid):
		return apperrors.ErrPaymentNotifyInvalid.WithMessage(err.Error())
	case errors.Is(err, payment.ErrCreateFailed):
		return apperrors.ErrCreatePaymentFailed.WithMessage(err.Error())
	case errors.Is(err, payment.ErrStatusInvalid):
		return apperrors.ErrPaymentStatusInvalid.WithMessage(err.Error())
	default:
		return err
	}
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
