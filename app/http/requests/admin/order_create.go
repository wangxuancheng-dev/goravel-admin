package admin

import (
	"goravel/app/http/trans"

	"github.com/goravel/framework/contracts/http"
)

// OrderCreateProduct 创建订单时的商品行。
type OrderCreateProduct struct {
	ProductID   uint    `form:"product_id" json:"product_id"`
	ProductName string  `form:"product_name" json:"product_name"`
	Price       float64 `form:"price" json:"price"`
	Quantity    int     `form:"quantity" json:"quantity"`
}

type OrderCreate struct {
	UserID    uint                 `form:"user_id" json:"user_id"`
	Amount    float64              `form:"amount" json:"amount"`
	Products  []OrderCreateProduct `form:"products" json:"products"`
	RequestID string               `form:"request_id" json:"request_id"`
	Remark    string               `form:"remark" json:"remark"`
}

func (r *OrderCreate) Authorize(ctx http.Context) error {
	return nil
}

func (r *OrderCreate) Rules(ctx http.Context) map[string]any {
	return map[string]any{
		"user_id":  "required",
		"amount":   "required",
		"products": "required",
	}
}

func (r *OrderCreate) Attributes(ctx http.Context) map[string]any {
	return map[string]any{
		"user_id":  trans.Get(ctx, "validation.attributes.user_id"),
		"amount":   trans.Get(ctx, "validation.attributes.amount"),
		"products": trans.Get(ctx, "validation.attributes.products"),
		"remark":   trans.Get(ctx, "validation.attributes.remark"),
	}
}
