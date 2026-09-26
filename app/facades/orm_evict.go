package facades

import (
	"reflect"
	"unsafe"

	contractsorm "github.com/goravel/framework/contracts/database/orm"
)

// EvictOrmConnectionCache drops a cached Orm query so the next Connection(name)
// rebuilds with the correct dbConfig.Database.
//
// Workaround for goravel/framework v1.18.0 Orm.Connection: on cache hit it
// reuses the parent Orm's dbConfig while returning the tenant query. HasTable /
// GetTables then inspect the platform database (table_schema=platform) while
// DML runs against the tenant database — tenant migrate becomes a no-op skip.
//
// Does not Close the underlying sql.DB: concurrent requests may still hold that
// pool. Forget() / Orm.Fresh() remain responsible for full teardown.
//
// Must hold ormConnectionMu: parallel migrate/seed call Evict while
// Orm/Schema.Connection mutates the same queries map under that lock.
func EvictOrmConnectionCache(name string) {
	if name == "" {
		return
	}
	ormConnectionMu.Lock()
	defer ormConnectionMu.Unlock()
	evictOrmConnectionCacheLocked(name)
}

// EvictOrmConnectionCacheLocked is EvictOrmConnectionCache when the caller
// already holds LockOrmConnectionBuild / ormConnectionMu.
func EvictOrmConnectionCacheLocked(name string) {
	if name == "" {
		return
	}
	evictOrmConnectionCacheLocked(name)
}

func evictOrmConnectionCacheLocked(name string) {
	o := Orm()
	if o == nil {
		return
	}
	// Unwrap tenant-aware router so we mutate the real Orm.queries map.
	if r, ok := o.(*ormRouter); ok {
		o = r.base
	}
	rv := reflect.ValueOf(o)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return
	}
	elem := reflect.NewAt(rv.Type().Elem(), unsafe.Pointer(rv.Pointer())).Elem()
	queriesField := elem.FieldByName("queries")
	if !queriesField.IsValid() || queriesField.Kind() != reflect.Map {
		return
	}
	// Must use the typed map via unsafe — MapIndex().Interface() panics on
	// values from unexported fields.
	m := *(*map[string]contractsorm.Query)(unsafe.Pointer(queriesField.UnsafeAddr()))
	delete(m, name)
}
