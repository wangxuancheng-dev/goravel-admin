package admin

import (
	"goravel/app/http/helpers"
	"goravel/app/http/trans"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/contracts/validation"
)

type AllowlistCreate struct {
	IP          string `form:"ip" json:"ip"`
	Remark      string `form:"remark" json:"remark"`
	Status      uint8  `form:"status" json:"status"`
	ConfirmCode string `form:"confirm_code" json:"confirm_code"`
}

func (r *AllowlistCreate) Authorize(ctx http.Context) error { return nil }

func (r *AllowlistCreate) Rules(ctx http.Context) map[string]any {
	return map[string]any{"ip": "required", "status": "in:0,1"}
}

func (r *AllowlistCreate) Attributes(ctx http.Context) map[string]any {
	return map[string]any{
		"ip":     trans.Get(ctx, "validation.attributes.ip"),
		"status": trans.Get(ctx, "validation.attributes.status"),
	}
}

func (r *AllowlistCreate) PrepareForValidation(ctx http.Context, data validation.Data) error {
	return helpers.PrepareNumericFieldForValidation(data, "status")
}
