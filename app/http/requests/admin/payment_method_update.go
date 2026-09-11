package admin

import (
	"goravel/app/http/trans"

	"github.com/goravel/framework/contracts/http"
)

type PaymentMethodUpdate struct {
	Name        *string        `form:"name" json:"name"`
	Config      map[string]any `form:"config" json:"config"`
	IsActive    *bool          `form:"is_active" json:"is_active"`
	Sort        *int           `form:"sort" json:"sort"`
	Description *string        `form:"description" json:"description"`
}

func (r *PaymentMethodUpdate) Authorize(ctx http.Context) error {
	return nil
}

func (r *PaymentMethodUpdate) Rules(ctx http.Context) map[string]any {
	return map[string]any{
		"name":      "max:50",
		"is_active": "boolean",
		"sort":      "min:0",
	}
}

func (r *PaymentMethodUpdate) Attributes(ctx http.Context) map[string]any {
	return map[string]any{
		"name":      trans.Get(ctx, "validation.attributes.name"),
		"is_active": trans.Get(ctx, "validation.attributes.is_active"),
		"sort":      trans.Get(ctx, "validation.attributes.sort"),
	}
}
