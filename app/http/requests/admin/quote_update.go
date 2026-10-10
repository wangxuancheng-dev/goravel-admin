package admin

import (
	"goravel/app/http/trans"

	"github.com/goravel/framework/contracts/http"
)

type QuoteUpdate struct {
	QuoteNo      *string `form:"quote_no" json:"quote_no"`
	CustomerName *string `form:"customer_name" json:"customer_name"`
	Status       *uint8  `form:"status" json:"status"`
	Remark       *string `form:"remark" json:"remark"`

	Details *[]QuoteItemInput `form:"details" json:"details"`
}

func (r *QuoteUpdate) Authorize(ctx http.Context) error {
	return nil
}

func (r *QuoteUpdate) Rules(ctx http.Context) map[string]any {
	rules := map[string]any{
		"quote_no":      "",
		"customer_name": "",
		"status":        "",
		"remark":        "",
	}
	return rules
}

func (r *QuoteUpdate) Attributes(ctx http.Context) map[string]any {
	return map[string]any{
		"quote_no":      trans.Get(ctx, "validation.attributes.quote_no"),
		"customer_name": trans.Get(ctx, "validation.attributes.customer_name"),
		"status":        trans.Get(ctx, "validation.attributes.status"),
		"remark":        trans.Get(ctx, "validation.attributes.remark"),
	}
}
