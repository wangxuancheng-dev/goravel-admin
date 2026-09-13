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
func EvictOrmConnectionCache(name string) {
	if name == "" {
		return
	}
	o := Orm()
	if o == nil {
		return
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
	key := reflect.ValueOf(name)
	if qv := queriesField.MapIndex(key); qv.IsValid() && !qv.IsZero() {
		if q, ok := qv.Interface().(contractsorm.Query); ok && q != nil {
			if db, err := q.DB(); err == nil && db != nil {
				_ = db.Close()
			}
		}
	}
	m := *(*map[string]contractsorm.Query)(unsafe.Pointer(queriesField.UnsafeAddr()))
	delete(m, name)
}
