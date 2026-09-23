package api

import (
	"strings"

	"github.com/goravel/framework/contracts/http"

	apperrors "goravel/app/errors"
	"goravel/app/http/response"
	"goravel/app/models"
	apppayment "goravel/app/payment"
	"goravel/app/services"
	"goravel/app/tenancy"
)

// PaymentNotifyController handles public payment callbacks for any registered gateway type.
// Route: POST /api/payment/notify/{type}[/{tenant}]
type PaymentNotifyController struct{}

func NewPaymentNotifyController() *PaymentNotifyController {
	return &PaymentNotifyController{}
}

// Notify handles /api/payment/notify/{type}/{tenant}
func (c *PaymentNotifyController) Notify(ctx http.Context) http.Response {
	typ := strings.TrimSpace(ctx.Request().Route("type"))
	tenant := strings.TrimSpace(ctx.Request().Route("tenant"))
	return c.handle(ctx, typ, tenant)
}

// NotifyLegacy handles /api/payment/notify/{type} (tenancy off only)
func (c *PaymentNotifyController) NotifyLegacy(ctx http.Context) http.Response {
	typ := strings.TrimSpace(ctx.Request().Route("type"))
	return c.handleLegacy(ctx, typ)
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
	typ = strings.ToLower(strings.TrimSpace(typ))
	if typ == "" {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrInvalidPaymentType)
	}
	if _, ok := apppayment.LookupGateway(typ); !ok {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrInvalidPaymentType)
	}
	if !services.IsPaymentGatewayEnabled(typ) {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrPaymentGatewayDisabled)
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

	method := c.resolvePaymentMethod(ctx, typ, data)
	payment, err := services.NewPaymentGatewayService(ctx).HandlePaymentNotify(method, data)
	if err == nil {
		return response.Success(ctx, map[string]any{
			"payment_no": payment.PaymentNo,
			"status":     payment.Status,
			"order_no":   payment.OrderNo,
		})
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

func (c *PaymentNotifyController) resolvePaymentMethod(ctx http.Context, typ string, data map[string]any) *models.PaymentMethod {
	svc := services.NewPaymentMethodService(ctx)
	if paymentNo := firstNotifyPaymentNo(data); paymentNo != "" {
		if payment, err := services.NewPaymentService(ctx).GetPaymentByPaymentNo(paymentNo); err == nil && payment != nil {
			if pm, err := svc.GetPaymentMethodByID(payment.PaymentMethodID); err == nil && pm != nil {
				return pm
			}
		}
	}
	if pm, err := svc.FindActiveByType(typ); err == nil && pm != nil {
		return pm
	}
	return &models.PaymentMethod{Type: typ, IsActive: true, Config: "{}"}
}

func firstNotifyPaymentNo(data map[string]any) string {
	for _, k := range []string{"out_trade_no", "payment_no"} {
		if s := strings.TrimSpace(stringFromAny(data[k])); s != "" {
			return s
		}
	}
	return ""
}

func stringFromAny(v any) string {
	switch t := v.(type) {
	case string:
		return t
	default:
		return ""
	}
}
