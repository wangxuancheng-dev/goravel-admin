package admin

import (
<<- $hasRichText := false>>
<<- range .FormFields>>
<<- if and (ne .Name "id") (ne .Name "created_at") (ne .Name "updated_at") (ne .Name "deleted_at") (or (eq .FormType "editor") (eq .FormType "markdown") (eq .FormType "richtext"))>>
<<- $hasRichText = true>>
<<- end>>
<<- end>>
<<- if $hasRichText>>
	"goravel/app/http/helpers"
<<- end>>
	"goravel/app/http/trans"

	"github.com/goravel/framework/contracts/http"
<<- if $hasRichText>>
	"github.com/goravel/framework/contracts/validation"
<<- end>>
)

<<if .IsMasterDetail>>
type <<.DetailModelName>>Input struct {
<<- range .DetailFormFields>>
	<<.FieldName>> <<.GoType>> `form:"<<.JsonName>>" json:"<<.JsonName>>"`
<<- end>>
}
<<end>>

type <<.RequestCreateName>> struct {
<<- range .FormFields>>
<<- if and (ne .Name "id") (ne .Name "created_at") (ne .Name "updated_at") (ne .Name "deleted_at")>>
	<<.FieldName>> <<.GoType>> `form:"<<.JsonName>>" json:"<<.JsonName>>"`
<<- end>>
<<- end>>
<<if .IsMasterDetail>>
	Details []<<.DetailModelName>>Input `form:"details" json:"details"`
<<end>>
}

func (r *<<.RequestCreateName>>) Authorize(ctx http.Context) error {
	return nil
}

func (r *<<.RequestCreateName>>) Rules(ctx http.Context) map[string]any {
	rules := map[string]any{
<<- range .FormFields>>
<<- if and (ne .Name "id") (ne .Name "created_at") (ne .Name "updated_at") (ne .Name "deleted_at")>>
		"<<.JsonName>>": "<<if .Required>>required<<end>><<if and .Required .Validators>>|<<end>><<range $i, $v := .Validators>><<if $i>>|<<end>><<$v>><<end>>",
<<- end>>
<<- end>>
	}
	return rules
}

func (r *<<.RequestCreateName>>) Attributes(ctx http.Context) map[string]any {
	return map[string]any{
<<- range .FormFields>>
<<- if and (ne .Name "id") (ne .Name "created_at") (ne .Name "updated_at") (ne .Name "deleted_at")>>
		"<<.JsonName>>": trans.Get(ctx, "validation.attributes.<<.Name>>"),
<<- end>>
<<- end>>
	}
}
<<- if $hasRichText>>

func (r *<<.RequestCreateName>>) PrepareForValidation(ctx http.Context, data validation.Data) error {
<<- range .FormFields>>
<<- if and (ne .Name "id") (ne .Name "created_at") (ne .Name "updated_at") (ne .Name "deleted_at") (or (eq .FormType "editor") (eq .FormType "markdown") (eq .FormType "richtext"))>>
	if err := helpers.PrepareRichTextFieldForValidation(data, "<<.JsonName>>"); err != nil {
		return err
	}
<<- end>>
<<- end>>
	return nil
}
<<- end>>