package admin

import (
	"goravel/app/http/helpers"
	"goravel/app/http/trans"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/contracts/validation"
)

type AllowlistUpdate struct {
	IP     *string `form:"ip" json:"ip"`
	Remark *string `form:"remark" json:"remark"`
	Status *uint8  `form:"status" json:"status"`
}

func (r *AllowlistUpdate) Authorize(ctx http.Context) error { return nil }

func (r *AllowlistUpdate) Rules(ctx http.Context) map[string]any {
	return map[string]any{"status": "in:0,1"}
}

func (r *AllowlistUpdate) Attributes(ctx http.Context) map[string]any {
	return map[string]any{"status": trans.Get(ctx, "validation.attributes.status")}
}

func (r *AllowlistUpdate) PrepareForValidation(ctx http.Context, data validation.Data) error {
	return helpers.PrepareNumericFieldForValidation(data, "status")
}
