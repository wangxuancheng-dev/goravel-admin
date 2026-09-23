package facades

import (
	"context"
	"database/sql"

	contractsbinding "github.com/goravel/framework/contracts/binding"
	"github.com/goravel/framework/contracts/database"
	"github.com/goravel/framework/contracts/database/orm"
	"github.com/goravel/framework/contracts/foundation"
)

// ormRouter delegates to a goroutine-bound tenant Orm when set (parallel seed),
// otherwise to the framework singleton.
type ormRouter struct {
	base orm.Orm
}

func (r *ormRouter) active() orm.Orm {
	if o := BoundTenantOrm(); o != nil {
		return o
	}
	return r.base
}

// InstallTenantAwareOrm wraps the container Orm so facades.Orm() honors BindTenantOrm.
func InstallTenantAwareOrm(app foundation.Application) {
	if app == nil {
		return
	}
	base := app.MakeOrm()
	if base == nil {
		return
	}
	if _, ok := base.(*ormRouter); ok {
		return
	}
	app.Instance(contractsbinding.Orm, &ormRouter{base: base})
	app.Fresh(contractsbinding.Orm)
}

// TenantAwareOrmInstalled reports whether MakeOrm returns the parallel-seed router.
func TenantAwareOrmInstalled() bool {
	_, ok := Orm().(*ormRouter)
	return ok
}

func (r *ormRouter) Config() database.Config {
	return r.active().Config()
}

func (r *ormRouter) Connection(name string) orm.Orm {
	ormConnectionMu.Lock()
	defer ormConnectionMu.Unlock()
	return r.base.Connection(name)
}

func (r *ormRouter) DB() (*sql.DB, error) {
	return r.active().DB()
}

func (r *ormRouter) Factory() orm.Factory {
	return r.active().Factory()
}

func (r *ormRouter) DatabaseName() string {
	return r.active().DatabaseName()
}

func (r *ormRouter) Name() string {
	return r.active().Name()
}

func (r *ormRouter) Observe(model any, observer orm.Observer) {
	r.active().Observe(model, observer)
}

func (r *ormRouter) Query() orm.Query {
	return r.active().Query()
}

func (r *ormRouter) Fresh() {
	r.active().Fresh()
}

func (r *ormRouter) SetQuery(query orm.Query) {
	if o := BoundTenantOrm(); o != nil {
		o.SetQuery(query)
		return
	}
	r.base.SetQuery(query)
}

func (r *ormRouter) Transaction(txFunc func(tx orm.Query) error) error {
	return r.active().Transaction(txFunc)
}

func (r *ormRouter) WithContext(ctx context.Context) orm.Orm {
	return r.active().WithContext(ctx)
}

var _ orm.Orm = (*ormRouter)(nil)
