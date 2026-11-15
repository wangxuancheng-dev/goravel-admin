package facades

import (
	"context"
	"database/sql"
	"sync"

	contractsbinding "github.com/goravel/framework/contracts/binding"
	"github.com/goravel/framework/contracts/database"
	"github.com/goravel/framework/contracts/database/orm"
	"github.com/goravel/framework/contracts/foundation"
)

// ormRouter delegates to the Orm of the goroutine-bound tenant Schema when set
// (migrate / seed, see WithBoundTenantSchema), otherwise to the framework singleton.
type ormRouter struct {
	base orm.Orm
}

func (r *ormRouter) active() orm.Orm {
	if s := BoundTenantSchema(); s != nil {
		if o := s.Orm(); o != nil {
			return o
		}
	}
	return r.base
}

// InstallTenantAwareOrm wraps the container Orm so facades.Orm() honors the
// goroutine-bound tenant Schema and resolves tenant connection names via tenantdb.
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

// Connection routes tenant connection names to the app-owned registry, so they never
// enter the framework's connection cache; other names go to the framework.
func (r *ormRouter) Connection(name string) orm.Orm {
	return BuildOrmConnection(name)
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

// Fresh rebuilds the framework's connections, then lets OnOrmFresh listeners re-attach
// whatever they had installed on the replaced gorm instances.
func (r *ormRouter) Fresh() {
	r.active().Fresh()
	ormFreshMu.Lock()
	listeners := append([]func(){}, ormFreshListeners...)
	ormFreshMu.Unlock()
	for _, fn := range listeners {
		fn()
	}
}

var (
	ormFreshMu        sync.Mutex
	ormFreshListeners []func()
)

// OnOrmFresh registers fn to run after facades.Orm().Fresh(), which replaces the
// gorm instances (and with them any callbacks registered on them).
func OnOrmFresh(fn func()) {
	if fn == nil {
		return
	}
	ormFreshMu.Lock()
	defer ormFreshMu.Unlock()
	ormFreshListeners = append(ormFreshListeners, fn)
}

func (r *ormRouter) SetQuery(query orm.Query) {
	r.active().SetQuery(query)
}

func (r *ormRouter) Transaction(txFunc func(tx orm.Query) error) error {
	return r.active().Transaction(txFunc)
}

func (r *ormRouter) WithContext(ctx context.Context) orm.Orm {
	return r.active().WithContext(ctx)
}

var _ orm.Orm = (*ormRouter)(nil)
