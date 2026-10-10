package admin

import (
	"goravel/app/http/trans"

	"github.com/goravel/framework/contracts/http"
)

type QuoteItemInput struct {
	ProductName string  `form:"product_name" json:"product_name"`
	Quantity    int     `form:"quantity" json:"quantity"`
	UnitPrice   float64 `form:"unit_price" json:"unit_price"`
}

type QuoteCreate struct {
	QuoteNo      string `form:"quote_no" json:"quote_no"`
	CustomerName string `form:"customer_name" json:"customer_name"`
	Status       uint8  `form:"status" json:"status"`
	Remark       string `form:"remark" json:"remark"`

	Details []QuoteItemInput `form:"details" json:"details"`
}

func (r *QuoteCreate) Authorize(ctx http.Context) error {
	return nil
}

func (r *QuoteCreate) Rules(ctx http.Context) map[string]any {
	rules := map[string]any{
		"quote_no":      "required",
		"customer_name": "required",
		"status":        "required",
		"remark":        "",
	}
	return rules
}

func (r *QuoteCreate) Attributes(ctx http.Context) map[string]any {
	return map[string]any{
		"quote_no":      trans.Get(ctx, "validation.attributes.quote_no"),
		"customer_name": trans.Get(ctx, "validation.attributes.customer_name"),
		"status":        trans.Get(ctx, "validation.attributes.status"),
		"remark":        trans.Get(ctx, "validation.attributes.remark"),
	}
}
