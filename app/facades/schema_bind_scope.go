package facades

import (
	"fmt"

	"github.com/goravel/framework/contracts/database/schema"
)

// WithBoundTenantSchema runs fn with sch bound to the current goroutine, so
// facades.Schema() / facades.Orm() (migrations, seeders, OrmQuery without a tenant
// ctx) hit the tenant database — without touching any process-wide state that
// concurrent requests read.
//
// The previous binding of this goroutine is restored afterwards, so calls nest.
// Work fn hands to other goroutines is NOT covered: pass a tenant-bound ctx and use
// OrmQuery(ctx) there. sch must not be shared with other goroutines (the migrator
// mutates the Schema it runs on): use a fresh one, e.g. tenantdb.Conn.NewSchema.
func WithBoundTenantSchema(sch schema.Schema, fn func() error) error {
	if fn == nil {
		return nil
	}
	if sch == nil {
		return fmt.Errorf("tenant schema is nil")
	}
	if !TenantRoutersInstalled() {
		return fmt.Errorf("tenant-aware Schema/Orm routers are not installed (DatabaseServiceProvider.Boot)")
	}
	gid := goroutineID()
	prev, had := tenantSchemaBinds.Load(gid)
	bindSchema(gid, sch)
	defer func() {
		if had {
			bindSchema(gid, prev.(schema.Schema))
		} else {
			unbindSchema(gid)
		}
	}()
	return fn()
}

// TenantRoutersInstalled reports whether both the Schema and Orm routers are
// installed (DatabaseServiceProvider.Boot), i.e. goroutine binding is effective.
func TenantRoutersInstalled() bool {
	return TenantAwareSchemaInstalled() && TenantAwareOrmInstalled()
}
