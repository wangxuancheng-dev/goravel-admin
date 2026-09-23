package facades

import (
	"sync"

	"github.com/goravel/framework/contracts/database/orm"
	"github.com/goravel/framework/contracts/database/schema"
)

var tenantOrmBinds sync.Map // goid(int64) -> orm.Orm

// BindTenantOrm pins Orm for the current goroutine (parallel tenant seed/migrate).
func BindTenantOrm(o orm.Orm) {
	if o == nil {
		return
	}
	tenantOrmBinds.Store(goroutineID(), o)
}

// UnbindTenantOrm clears the current goroutine's tenant Orm pin.
func UnbindTenantOrm() {
	tenantOrmBinds.Delete(goroutineID())
}

// BoundTenantOrm returns the goroutine-local tenant Orm, if any.
func BoundTenantOrm() orm.Orm {
	if v, ok := tenantOrmBinds.Load(goroutineID()); ok {
		if o, ok := v.(orm.Orm); ok && o != nil {
			return o
		}
	}
	return nil
}

// BindTenantSchemaAndOrm pins Schema + Orm for parallel tenant migrate/seed.
func BindTenantSchemaAndOrm(sch schema.Schema) {
	BindTenantSchema(sch)
	if sch != nil {
		BindTenantOrm(sch.Orm())
	}
}

// UnbindTenantSchemaAndOrm clears Schema + Orm pins for this goroutine.
func UnbindTenantSchemaAndOrm() {
	UnbindTenantSchema()
	UnbindTenantOrm()
}
