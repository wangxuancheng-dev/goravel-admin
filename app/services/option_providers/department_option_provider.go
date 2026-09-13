package option_providers

import (
	"context"
	appfacades "goravel/app/facades"

	"github.com/goravel/framework/contracts/http"

	"goravel/app/models"
)

type DepartmentOptionProvider struct {
	ctx context.Context
}

func NewDepartmentOptionProvider(ctx context.Context) *DepartmentOptionProvider {
	return &DepartmentOptionProvider{
		ctx: ctx}
}

func (p *DepartmentOptionProvider) GetOptions(ctx http.Context) (map[string]any, error) {
	var departments []models.Department
	if err := appfacades.OrmQuery(ctx).Where("status", 1).Order("sort asc, id asc").Get(&departments); err != nil {
		return nil, err
	}

	tree := p.buildDepartmentTree(departments, 0)

	return map[string]any{
		"options": tree,
	}, nil
}

func (p *DepartmentOptionProvider) buildDepartmentTree(departments []models.Department, parentID uint) []map[string]any {
	var tree []map[string]any
	for _, dept := range departments {
		if dept.ParentID == parentID {
			node := map[string]any{
				"id":   dept.ID,
				"name": dept.Name,
			}
			children := p.buildDepartmentTree(departments, dept.ID)
			if len(children) > 0 {
				node["children"] = children
			}
			tree = append(tree, node)
		}
	}
	return tree
}
