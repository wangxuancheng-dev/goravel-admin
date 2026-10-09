package models

import (
	"github.com/goravel/framework/database/orm"
)

type <<.ModelName>> struct {
	orm.Model
<<range .Fields>>
<<- if and (ne .Name "id") (ne .Name "created_at") (ne .Name "updated_at") (ne .Name "deleted_at")>>
	<<if and .Relation (eq .Relation.RelationType "belongsTo")>>
	<<.FieldName>> uint `gorm:"column:<<.Name>>" json:"<<.JsonName>>"<<if .Comment>> comment:"<<.Comment>>"<<end>>`
	<<else>>
	<<.FieldName>> <<.GoType>> `gorm:"column:<<.Name>>" json:"<<.JsonName>>"<<if .Comment>> comment:"<<.Comment>>"<<end>>`
	<<end>>
<<- end>>
<<if and .Relation (or (eq .Relation.RelationType "belongsTo") (eq .Relation.RelationType "hasOne"))>>
	<<.Relation.Name>> *<<.Relation.ModelName>> `gorm:"foreignKey:<<.FieldName>>" json:"<<.Relation.JsonName>>"`
<<end>>
<<- end>>
<<if .IsTreeList>>
	Children []<<.ModelName>> `gorm:"-" json:"children,omitempty"`
<<end>>
<<if .IsMasterDetail>>
	Details []<<.DetailModelName>> `gorm:"foreignKey:<<.DetailFKFieldName>>" json:"details"`
<<end>>
	orm.SoftDeletes
}

func (<<.ModelName>>) TableName() string {
	return "<<.TableName>>"
}
