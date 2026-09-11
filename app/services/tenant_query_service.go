package services

import (
	appfacades "goravel/app/facades"
	"goravel/app/http/helpers"
	"goravel/app/tenancy"

	"github.com/goravel/framework/contracts/database/orm"
	"github.com/goravel/framework/contracts/http"
)

// TenantQueryService is a thin alias of OrmQuery for readability at call sites.
type TenantQueryService struct {
	ctx http.Context
}

func NewTenantQueryService(ctx http.Context) *TenantQueryService {
	return &TenantQueryService{ctx: ctx}
}

func (s *TenantQueryService) Query() orm.Query {
	return appfacades.OrmQuery(s.ctx)
}

func (s *TenantQueryService) QueryModel(model any) orm.Query {
	return appfacades.OrmQuery(s.ctx).Model(model)
}

func (s *TenantQueryService) QueryTable(tableName string) orm.Query {
	return appfacades.OrmQuery(s.ctx).Table(tableName)
}

func (s *TenantQueryService) GetTenantID() (uint, bool) {
	return helpers.GetTenantIDFromContext(s.ctx)
}

// TenantBound reports whether ctx is bound to a tenant connection.
func (s *TenantQueryService) TenantBound() bool {
	return tenancy.Bound(s.ctx)
}
