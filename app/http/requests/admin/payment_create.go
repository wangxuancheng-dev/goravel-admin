package admin

import (
	"goravel/app/http/trans"

	"github.com/goravel/framework/contracts/http"
)

// PaymentCreate creates a payment row for an existing pending order (reference checkout).
type PaymentCreate struct {
	OrderNo         string  `form:"order_no" json:"order_no"`
	PaymentMethodID uint    `form:"payment_method_id" json:"payment_method_id"`
	Amount          float64 `form:"amount" json:"amount"`
	Remark          string  `form:"remark" json:"remark"`
	Initiate        bool    `form:"initiate" json:"initiate"` // call gateway CreatePaymentOrder (mock returns notify hint)
}

func (r *PaymentCreate) Authorize(ctx http.Context) error {
	return nil
}

func (r *PaymentCreate) Rules(ctx http.Context) map[string]any {
	return map[string]any{
		"order_no":          "required|max:50",
		"payment_method_id": "required",
		"amount":            "min:0",
		"remark":            "max:500",
		"initiate":          "boolean",
	}
}

func (r *PaymentCreate) Messages(ctx http.Context) map[string]any {
	return map[string]any{}
}

func (r *PaymentCreate) Attributes(ctx http.Context) map[string]any {
	return map[string]any{
		"order_no":          trans.Get(ctx, "validation.attributes.order_no"),
		"payment_method_id": trans.Get(ctx, "validation.attributes.payment_method_id"),
		"amount":            trans.Get(ctx, "validation.attributes.amount"),
		"remark":            trans.Get(ctx, "validation.attributes.remark"),
	}
}
