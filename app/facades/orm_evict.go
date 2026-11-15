package facades

import (
	"reflect"
	"sync"
	"unsafe"

	contractsorm "github.com/goravel/framework/contracts/database/orm"
	"github.com/goravel/framework/facades"
)

// EvictOrmConnectionCache drops a cached Orm connection so the next Connection(name)
// rebuilds it from the current database.connections.<name> config.
//
// The framework Orm caches every connection it builds (query + dbConfig) by name and
// never invalidates it. Tenant connections are re-registered at runtime (DropStorage,
// connection-name reuse, idle eviction), and a stale entry would keep the old
// dbConfig.Database, so HasTable / DatabaseName checks inspect the wrong database.
// (framework <= v1.18.0 additionally returned the parent's dbConfig on every cache hit.)
//
// Does not Close the underlying sql.DB: concurrent requests may still hold that
// pool. Forget() / Orm.Fresh() remain responsible for full teardown.
//
// Must hold ormConnectionMu: parallel migrate/seed call Evict while
// Orm/Schema.Connection builds connections under that lock.
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

var evictUnsupportedOnce sync.Once

func evictOrmConnectionCacheLocked(name string) {
	o := Orm()
	if o == nil {
		return
	}
	if !evictFromOrm(o, name) {
		evictUnsupportedOnce.Do(func() {
			if l := facades.Log(); l != nil {
				l.Warningf("orm connection cache layout not recognized (framework upgrade?): tenant connection eviction is disabled; update app/facades/orm_evict.go")
			}
		})
	}
}

// evictFromOrm deletes name from the framework Orm's private connection cache.
// It reports false when the Orm's layout is not one it knows, so a framework
// upgrade that changes it is noticed (TestEvictFromOrm fails, a warning is logged)
// instead of eviction silently becoming a no-op.
//
// Known layouts:
//   - <= v1.18.0: queries map[string]orm.Query
//   - v1.18.1+:   queries *QueriesCache{mu sync.RWMutex; connections map[string]cachedConnection}
func evictFromOrm(o contractsorm.Orm, name string) bool {
	// Unwrap tenant-aware router so we mutate the real Orm cache.
	if r, ok := o.(*ormRouter); ok {
		o = r.base
	}
	rv := reflect.ValueOf(o)
	if !rv.IsValid() || rv.Kind() != reflect.Ptr || rv.IsNil() || rv.Elem().Kind() != reflect.Struct {
		return false
	}
	field := rv.Elem().FieldByName("queries")
	if !field.IsValid() {
		return false
	}
	queries := writable(field)

	switch queries.Kind() {
	case reflect.Map:
		return deleteMapKey(queries, name)
	case reflect.Ptr:
		if queries.IsNil() {
			return true
		}
		cache := queries.Elem()
		if cache.Kind() != reflect.Struct {
			return false
		}
		muField := cache.FieldByName("mu")
		connsField := cache.FieldByName("connections")
		if !muField.IsValid() || muField.Type() != reflect.TypeOf(sync.RWMutex{}) ||
			!connsField.IsValid() || connsField.Kind() != reflect.Map {
			return false
		}
		mu := (*sync.RWMutex)(unsafe.Pointer(muField.UnsafeAddr()))
		mu.Lock()
		defer mu.Unlock()
		return deleteMapKey(writable(connsField), name)
	default:
		return false
	}
}

// writable returns an addressable, settable view of an (unexported) struct field.
func writable(field reflect.Value) reflect.Value {
	return reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem()
}

func deleteMapKey(m reflect.Value, name string) bool {
	if m.Type().Key().Kind() != reflect.String {
		return false
	}
	if m.IsNil() {
		return true
	}
	m.SetMapIndex(reflect.ValueOf(name).Convert(m.Type().Key()), reflect.Value{})
	return true
}
