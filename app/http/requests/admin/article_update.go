package admin

import (
	"goravel/app/http/helpers"
	"goravel/app/http/trans"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/contracts/validation"
)

type ArticleUpdate struct {
	AdminId *int64  `form:"admin_id" json:"admin_id"`
	Title   *string `form:"title" json:"title"`
	Content *string `form:"content" json:"content"`
	Status  *uint8  `form:"status" json:"status"`
}

func (r *ArticleUpdate) Authorize(ctx http.Context) error {
	return nil
}

func (r *ArticleUpdate) Rules(ctx http.Context) map[string]any {
	rules := map[string]any{
		"admin_id": "",
		"title":    "",
		"content":  "",
		"status":   "",
	}
	return rules
}

func (r *ArticleUpdate) Attributes(ctx http.Context) map[string]any {
	return map[string]any{
		"admin_id": trans.Get(ctx, "validation.attributes.admin_id"),
		"title":    trans.Get(ctx, "validation.attributes.title"),
		"content":  trans.Get(ctx, "validation.attributes.content"),
		"status":   trans.Get(ctx, "validation.attributes.status"),
	}
}

func (r *ArticleUpdate) PrepareForValidation(ctx http.Context, data validation.Data) error {
	if err := helpers.PrepareRichTextFieldForValidation(data, "content"); err != nil {
		return err
	}
	return nil
}
