package api

import (
	"strings"

	"github.com/goravel/framework/contracts/http"

	apperrors "goravel/app/errors"
	"goravel/app/http/response"
	"goravel/app/models"
	"goravel/app/services"
)

// PaymentNotifyController exposes scaffold notify endpoints that intentionally
// return payment_gateway_not_implemented until real gateway wiring is added.
type PaymentNotifyController struct{}

func NewPaymentNotifyController() *PaymentNotifyController {
	return &PaymentNotifyController{}
}

// Notify handles POST /api/payment/notify/{type} where type is wechat|alipay.
func (c *PaymentNotifyController) Notify(ctx http.Context) http.Response {
	typ := strings.ToLower(strings.TrimSpace(ctx.Request().Route("type")))
	if typ != "wechat" && typ != "alipay" {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrInvalidPaymentType)
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
	return response.ErrorWithLog(ctx, "payment_notify", err, map[string]any{"type": typ})
}
