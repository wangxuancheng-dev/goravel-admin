package admin

import (
	"goravel/app/http/helpers"
	"goravel/app/http/trans"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/contracts/validation"
)

type BlacklistCreate struct {
	IP          string `form:"ip" json:"ip" example:"192.168.1.1"`           // IP地址/IP段（支持单IP、CIDR、范围）
	Remark      string `form:"remark" json:"remark" example:"测试IP"`          // 备注说明（可选）
	Status      uint8  `form:"status" json:"status" enums:"0,1" example:"1"` // 状态（1-启用，0-禁用）
	ConfirmCode string `form:"confirm_code" json:"confirm_code"`             // 敏感操作二次确认（TOTP 或当前密码）
}

func (r *BlacklistCreate) Authorize(ctx http.Context) error {
	return nil
}

func (r *BlacklistCreate) Rules(ctx http.Context) map[string]any {
	return map[string]any{
		"ip":     "required",
		"status": "in:0,1",
	}
}

func (r *BlacklistCreate) Attributes(ctx http.Context) map[string]any {
	return map[string]any{
		"ip":     trans.Get(ctx, "validation.attributes.ip"),
		"status": trans.Get(ctx, "validation.attributes.status"),
	}
}

func (r *BlacklistCreate) PrepareForValidation(ctx http.Context, data validation.Data) error {
	return helpers.PrepareNumericFieldForValidation(data, "status")
}
