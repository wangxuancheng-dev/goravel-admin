package facades

import (
	"fmt"

	"github.com/goravel/framework/contracts/database/schema"
)

// WithBoundTenantSchema runs fn with sch (and sch.Orm()) bound to the current
// goroutine, so facades.Schema() / facades.Orm() and code built on them (migrations,
// seeders, OrmQuery without a tenant ctx) hit the tenant connection — without
// touching any process-wide state that concurrent requests read.
//
// The previous binding of this goroutine is restored afterwards, so calls nest.
// Work fn hands to other goroutines is NOT covered: pass a tenant-bound ctx and use
// OrmQuery(ctx) there.
func WithBoundTenantSchema(sch schema.Schema, fn func() error) error {
	if fn == nil {
		return nil
	}
	if sch == nil {
		return fmt.Errorf("tenant schema is nil")
	}
	gid := goroutineID()
	prevSchema, hadSchema := tenantSchemaBinds.Load(gid)
	prevOrm, hadOrm := tenantOrmBinds.Load(gid)

	tenantSchemaBinds.Store(gid, sch)
	if o := sch.Orm(); o != nil {
		tenantOrmBinds.Store(gid, o)
	} else {
		tenantOrmBinds.Delete(gid)
	}
	defer func() {
		if hadSchema {
			tenantSchemaBinds.Store(gid, prevSchema)
		} else {
			tenantSchemaBinds.Delete(gid)
		}
		if hadOrm {
			tenantOrmBinds.Store(gid, prevOrm)
		} else {
			tenantOrmBinds.Delete(gid)
		}
	}()

	return fn()
}

// TenantRoutersInstalled reports whether both the Schema and Orm routers are
// installed (DatabaseServiceProvider.Boot), i.e. goroutine binding is effective.
func TenantRoutersInstalled() bool {
	return TenantAwareSchemaInstalled() && TenantAwareOrmInstalled()
}
