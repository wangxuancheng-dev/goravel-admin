package api

import (
	"strings"

	"github.com/goravel/framework/contracts/http"

	apperrors "goravel/app/errors"
	"goravel/app/http/response"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/app/tenancy"
)

// PaymentNotifyController exposes scaffold notify endpoints that intentionally
// return payment_gateway_not_implemented until real gateway wiring is added.
type PaymentNotifyController struct{}

func NewPaymentNotifyController() *PaymentNotifyController {
	return &PaymentNotifyController{}
}

func (c *PaymentNotifyController) NotifyWechat(ctx http.Context) http.Response {
	return c.handle(ctx, "wechat", strings.TrimSpace(ctx.Request().Route("tenant")))
}

func (c *PaymentNotifyController) NotifyAlipay(ctx http.Context) http.Response {
	return c.handle(ctx, "alipay", strings.TrimSpace(ctx.Request().Route("tenant")))
}

func (c *PaymentNotifyController) NotifyLegacyWechat(ctx http.Context) http.Response {
	return c.handleLegacy(ctx, "wechat")
}

func (c *PaymentNotifyController) NotifyLegacyAlipay(ctx http.Context) http.Response {
	return c.handleLegacy(ctx, "alipay")
}

func (c *PaymentNotifyController) handleLegacy(ctx http.Context, typ string) http.Response {
	if tenancy.Enabled() {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrTenantRequired.WithMessage(
			"payment notify requires /api/payment/notify/"+typ+"/{tenant_code}",
		))
	}
	return c.handle(ctx, typ, "")
}

func (c *PaymentNotifyController) handle(ctx http.Context, typ, tenantCode string) http.Response {
	if typ != "wechat" && typ != "alipay" {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrInvalidPaymentType)
	}

	if tenancy.Enabled() {
		if tenantCode == "" {
			return response.Error(ctx, http.StatusBadRequest, apperrors.ErrTenantRequired)
		}
		if err := services.NewTenantConnectionService().BindHTTP(ctx, tenantCode); err != nil {
			if businessErr, ok := apperrors.GetBusinessError(err); ok {
				status := http.StatusBadRequest
				switch businessErr.Code {
				case apperrors.ErrTenantNotFound.Code:
					status = http.StatusNotFound
				case apperrors.ErrTenantDisabled.Code, apperrors.ErrTenantNotReady.Code:
					status = http.StatusForbidden
				case apperrors.ErrTenantHintConflict.Code:
					status = http.StatusBadRequest
				}
				return response.Error(ctx, status, businessErr)
			}
			return response.Error(ctx, http.StatusBadGateway, apperrors.ErrTenantConnectionFailed)
		}
	}

	data := ctx.Request().All()
	if data == nil {
		data = map[string]any{}
	}

	_, err := services.NewPaymentGatewayService(ctx).HandlePaymentNotify(
		&models.PaymentMethod{Type: typ, IsActive: true},
		data,
	)
	if err == nil {
		return response.Success(ctx)
	}
	if businessErr, ok := apperrors.GetBusinessError(err); ok {
		status := http.StatusBadRequest
		if businessErr.Code == apperrors.ErrPaymentGatewayNotImplemented.Code {
			status = http.StatusNotImplemented
		}
		return response.Error(ctx, status, businessErr)
	}
	return response.ErrorWithLog(ctx, "payment_notify", err, map[string]any{"type": typ, "tenant": tenantCode})
}
