package admin

import (
	"goravel/app/http/helpers"
	"goravel/app/http/trans"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/contracts/validation"
)

type MenuUpdate struct {
	ParentID   *uint   `form:"parent_id" json:"parent_id"`
	Title      *string `form:"title" json:"title"`
	Slug       *string `form:"slug" json:"slug"`
	Icon       *string `form:"icon" json:"icon"`
	Path       *string `form:"path" json:"path"`
	Component  *string `form:"component" json:"component"`
	Permission *string `form:"permission" json:"permission"`
	Type       *uint8  `form:"type" json:"type"`
	Status     *uint8  `form:"status" json:"status"`
	Sort       *int    `form:"sort" json:"sort"`
	IsHidden   *uint8  `form:"is_hidden" json:"is_hidden"`
	LinkType   *uint8  `form:"link_type" json:"link_type"`
	OpenType   *uint8  `form:"open_type" json:"open_type"`
	NoCache    *uint8  `form:"no_cache" json:"no_cache"`
}

func (r *MenuUpdate) Authorize(ctx http.Context) error {
	return nil
}

func (r *MenuUpdate) Rules(ctx http.Context) map[string]any {
	rules := map[string]any{
		"title":      "max:50",
		"slug":       "max:50",
		"icon":       "max:50",
		"component":  "max:255",
		"permission": "max:100",
		"type":       "in:1,2,3",
		"status":     "in:0,1",
		"is_hidden":  "in:0,1",
		"link_type":  "in:1,2",
		"open_type":  "in:1,2",
		"no_cache":   "in:0,1",
	}

	linkType := ctx.Request().Input("link_type")
	if linkType == "2" {
		rules["path"] = "max:1000|url"
	} else {
		rules["path"] = "max:1000"
		delete(rules, "open_type")
	}

	return rules
}

func (r *MenuUpdate) Attributes(ctx http.Context) map[string]any {
	return map[string]any{
		"title":      trans.Get(ctx, "validation.attributes.title"),
		"slug":       trans.Get(ctx, "validation.attributes.slug"),
		"icon":       trans.Get(ctx, "validation.attributes.icon"),
		"path":       trans.Get(ctx, "validation.attributes.path"),
		"component":  trans.Get(ctx, "validation.attributes.component"),
		"permission": trans.Get(ctx, "validation.attributes.permission"),
		"type":       trans.Get(ctx, "validation.attributes.type"),
		"status":     trans.Get(ctx, "validation.attributes.status"),
		"is_hidden":  trans.Get(ctx, "validation.attributes.is_hidden"),
	}
}

func (r *MenuUpdate) PrepareForValidation(ctx http.Context, data validation.Data) error {
	if err := helpers.PrepareNumericFieldForValidation(data, "type"); err != nil {
		return err
	}
	if err := helpers.PrepareNumericFieldForValidation(data, "status"); err != nil {
		return err
	}
	if err := helpers.PrepareNumericFieldForValidation(data, "is_hidden"); err != nil {
		return err
	}
	if err := helpers.PrepareNumericFieldForValidation(data, "link_type"); err != nil {
		return err
	}
	if err := helpers.PrepareNumericFieldForValidation(data, "open_type"); err != nil {
		return err
	}
	if err := helpers.PrepareNumericFieldForValidation(data, "no_cache"); err != nil {
		return err
	}
	return nil
}
