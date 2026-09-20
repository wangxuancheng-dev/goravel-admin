package admin

import (
	"goravel/app/http/trans"

	"github.com/goravel/framework/contracts/http"
)

type ResetPassword struct {
	Password        string `form:"password" json:"password"`
	ConfirmCode     string `form:"confirm_code" json:"confirm_code"`
	ConfirmPassword string `form:"confirm_password" json:"confirm_password"`
}

func (r *ResetPassword) Authorize(ctx http.Context) error {
	return nil
}

func (r *ResetPassword) Rules(ctx http.Context) map[string]any {
	// Strength is enforced by login_security policy in AdminService.ResetPassword.
	return map[string]any{
		"password": "required",
	}
}

func (r *ResetPassword) Attributes(ctx http.Context) map[string]any {
	return map[string]any{
		"password": trans.Get(ctx, "validation.attributes.password"),
	}
}

// SensitiveConfirmCode returns confirm_code, falling back to confirm_password.
func (r *ResetPassword) SensitiveConfirmCode() string {
	if r.ConfirmCode != "" {
		return r.ConfirmCode
	}
	return r.ConfirmPassword
}
