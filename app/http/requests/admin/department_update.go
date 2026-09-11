package admin

import (
	"goravel/app/http/helpers"
	"goravel/app/http/trans"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/contracts/validation"
)

type DepartmentUpdate struct {
	ParentID *uint   `form:"parent_id" json:"parent_id"`
	Name     *string `form:"name" json:"name"`
	Code     *string `form:"code" json:"code"`
	Leader   *string `form:"leader" json:"leader"`
	Phone    *string `form:"phone" json:"phone"`
	Email    *string `form:"email" json:"email"`
	Status   *uint8  `form:"status" json:"status"`
	Sort     *int    `form:"sort" json:"sort"`
	Remark   *string `form:"remark" json:"remark"`
}

func (r *DepartmentUpdate) Authorize(ctx http.Context) error {
	return nil
}

func (r *DepartmentUpdate) Rules(ctx http.Context) map[string]any {
	return map[string]any{
		"name":   "max:50",
		"code":   "max:50",
		"leader": "max:50",
		"phone":  "max:20",
		"email":  "email|max:100",
		"status": "in:0,1",
		"remark": "max:500",
	}
}

func (r *DepartmentUpdate) Attributes(ctx http.Context) map[string]any {
	return map[string]any{
		"name":   trans.Get(ctx, "validation.attributes.name"),
		"code":   trans.Get(ctx, "validation.attributes.code"),
		"leader": trans.Get(ctx, "validation.attributes.leader"),
		"phone":  trans.Get(ctx, "validation.attributes.phone"),
		"email":  trans.Get(ctx, "validation.attributes.email"),
		"status": trans.Get(ctx, "validation.attributes.status"),
		"remark": trans.Get(ctx, "validation.attributes.remark"),
	}
}

func (r *DepartmentUpdate) PrepareForValidation(ctx http.Context, data validation.Data) error {
	return helpers.PrepareNumericFieldForValidation(data, "status")
}
