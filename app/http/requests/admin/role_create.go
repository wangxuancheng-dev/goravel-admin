package admin

import (
	"goravel/app/http/helpers"
	"goravel/app/http/trans"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/contracts/validation"
)

type RoleCreate struct {
	Name        string `form:"name" json:"name"`
	Slug        string `form:"slug" json:"slug"`
	Description string `form:"description" json:"description"`
	Status      uint8  `form:"status" json:"status"`
	Sort        int    `form:"sort" json:"sort"`
	DataScope   uint8  `form:"data_scope" json:"data_scope"`
}

func (r *RoleCreate) Authorize(ctx http.Context) error {
	return nil
}

func (r *RoleCreate) Rules(ctx http.Context) map[string]any {
	return map[string]any{
		"name":        "required|max:50",
		"slug":        "required|max:50",
		"description": "max:255",
		"status":      "in:0,1",
		"data_scope":  "in:1,2,3,4,5",
	}
}

func (r *RoleCreate) Attributes(ctx http.Context) map[string]any {
	return map[string]any{
		"name":        trans.Get(ctx, "validation.attributes.name"),
		"slug":        trans.Get(ctx, "validation.attributes.slug"),
		"description": trans.Get(ctx, "validation.attributes.description"),
		"status":      trans.Get(ctx, "validation.attributes.status"),
		"data_scope":  trans.Get(ctx, "validation.attributes.data_scope"),
	}
}

func (r *RoleCreate) PrepareForValidation(ctx http.Context, data validation.Data) error {
	if err := helpers.PrepareNumericFieldForValidation(data, "status"); err != nil {
		return err
	}
	return helpers.PrepareNumericFieldForValidation(data, "data_scope")
}
