package jobs

import (
	"context"

	"github.com/goravel/framework/facades"
	"github.com/spf13/cast"

	apperrors "goravel/app/errors"
	"goravel/app/services"
	"goravel/app/tenancy"
)

// CancelExpiredOrder cancels a pending order after Delay(expire_at). Idempotent.
type CancelExpiredOrder struct{}

func (r *CancelExpiredOrder) Signature() string {
	return "cancel_expired_order"
}

func (r *CancelExpiredOrder) Handle(args ...any) error {
	if len(args) < 1 {
		return apperrors.ErrInvalidArgument.WithMessage("missing order_no")
	}
	orderNo := cast.ToString(args[0])
	if orderNo == "" {
		return apperrors.ErrOrderNoRequired
	}
	tenantID := uint(0)
	if len(args) >= 2 {
		tenantID = cast.ToUint(args[1])
	}

	ctx := context.Background()
	if tenancy.Enabled() {
		if tenantID == 0 {
			return apperrors.ErrTenantRequired
		}
		bound, err := services.NewTenantConnectionService().BindBackground(ctx, tenantID)
		if err != nil {
			facades.Log().Errorf("CancelExpiredOrder bind tenant failed: tenant_id=%d err=%v", tenantID, err)
			return err
		}
		ctx = bound
	}

	cancelled, err := services.CancelOrderIfExpired(ctx, orderNo)
	if err != nil {
		facades.Log().Errorf("CancelExpiredOrder failed: order_no=%s err=%v", orderNo, err)
		return err
	}
	if cancelled {
		facades.Log().Infof("CancelExpiredOrder cancelled: order_no=%s", orderNo)
	}
	return nil
}
