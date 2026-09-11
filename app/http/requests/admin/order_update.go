package admin

import (
	"goravel/app/http/trans"

	"github.com/goravel/framework/contracts/http"
)

type OrderUpdate struct {
	OrderNo string `form:"order_no" json:"order_no"`
	Status  string `form:"status" json:"status"`
	Remark  string `form:"remark" json:"remark"`
}

func (r *OrderUpdate) Authorize(ctx http.Context) error {
	return nil
}

func (r *OrderUpdate) Rules(ctx http.Context) map[string]any {
	return map[string]any{
		"status": "required|in:pending,paid,cancelled",
	}
}

func (r *OrderUpdate) Attributes(ctx http.Context) map[string]any {
	return map[string]any{
		"order_no": trans.Get(ctx, "validation.attributes.order_no"),
		"status":   trans.Get(ctx, "validation.attributes.status"),
		"remark":   trans.Get(ctx, "validation.attributes.remark"),
	}
}
