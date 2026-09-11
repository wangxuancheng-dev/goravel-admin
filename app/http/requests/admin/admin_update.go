package admin

import (
	"goravel/app/http/helpers"
	"goravel/app/http/trans"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/contracts/validation"
)

type AdminUpdate struct {
	Nickname     *string `form:"nickname" json:"nickname" example:"管理员"`                             // 显示昵称（可选，最大50字符）
	Email        *string `form:"email" json:"email" example:"admin@example.com"`                     // 联系邮箱（可选，需符合邮箱格式）
	Phone        *string `form:"phone" json:"phone" example:"13800138000"`                           // 联系手机号（可选，最大20字符）
	Password     *string `form:"password" json:"password" example:"123456"`                          // 登录密码（可选，6-50字符，不传则不修改）
	DepartmentID *uint   `form:"department_id" json:"department_id" example:"1"`                     // 所属部门ID（可选，关联部门表主键）
	PositionID   *uint   `form:"position_id" json:"position_id" example:"1"`                         // 所属岗位ID（可选，关联岗位表主键）
	Status       *uint8  `form:"status" json:"status" enums:"0,1" example:"1"`                       // 账号状态（可选，1-启用，0-禁用）
	RoleIDs      []uint  `form:"role_ids" json:"role_ids" swaggertype:"array,integer" example:"1,2"` // 角色ID数组（可选；是否同步看请求是否带 role_ids）
}

func (r *AdminUpdate) Authorize(ctx http.Context) error {
	return nil
}

func (r *AdminUpdate) Rules(ctx http.Context) map[string]any {
	return map[string]any{
		"nickname": "max:50",
		"email":    "email|max:100",
		"phone":    "max:20",
		"password": "min:6|max:50",
		"status":   "in:0,1",
	}
}

func (r *AdminUpdate) Attributes(ctx http.Context) map[string]any {
	return map[string]any{
		"nickname": trans.Get(ctx, "validation.attributes.nickname"),
		"email":    trans.Get(ctx, "validation.attributes.email"),
		"phone":    trans.Get(ctx, "validation.attributes.phone"),
		"password": trans.Get(ctx, "validation.attributes.password"),
		"status":   trans.Get(ctx, "validation.attributes.status"),
	}
}

func (r *AdminUpdate) PrepareForValidation(ctx http.Context, data validation.Data) error {
	return helpers.PrepareNumericFieldForValidation(data, "status")
}
