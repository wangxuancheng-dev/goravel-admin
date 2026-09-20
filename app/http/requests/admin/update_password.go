package admin

import (
	"goravel/app/http/trans"

	"github.com/goravel/framework/contracts/http"
)

type UpdatePassword struct {
	OldPassword     string `form:"old_password" json:"old_password"`
	NewPassword     string `form:"new_password" json:"new_password"`
	ConfirmPassword string `form:"confirm_password" json:"confirm_password"`
}

func (r *UpdatePassword) Authorize(ctx http.Context) error {
	return nil
}

func (r *UpdatePassword) Rules(ctx http.Context) map[string]any {
	// Strength (min length / letter / number / special) is enforced by login_security policy in service layer.
	return map[string]any{
		"old_password":     "required",
		"new_password":     "required",
		"confirm_password": "required|same:new_password",
	}
}

func (r *UpdatePassword) Attributes(ctx http.Context) map[string]any {
	return map[string]any{
		"old_password":     trans.Get(ctx, "validation.attributes.old_password"),
		"new_password":     trans.Get(ctx, "validation.attributes.new_password"),
		"confirm_password": trans.Get(ctx, "validation.attributes.confirm_password"),
	}
}
