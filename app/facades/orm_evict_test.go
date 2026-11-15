package facades

import (
	"context"
	"reflect"
	"sort"
	"testing"

	"github.com/goravel/framework/contracts/database"
	contractsorm "github.com/goravel/framework/contracts/database/orm"
	frameworkorm "github.com/goravel/framework/database/orm"
)

// cachedConnectionNames reads the framework Orm cache keys (test helper).
func cachedConnectionNames(t *testing.T, o contractsorm.Orm) []string {
	t.Helper()
	queries := writable(reflect.ValueOf(o).Elem().FieldByName("queries"))
	conns := queries
	if queries.Kind() == reflect.Ptr {
		conns = writable(queries.Elem().FieldByName("connections"))
	}
	names := make([]string, 0, conns.Len())
	for _, k := range conns.MapKeys() {
		names = append(names, k.String())
	}
	sort.Strings(names)
	return names
}

// Guards against a framework upgrade changing the Orm cache layout again
// (v1.18.1 turned the map into *QueriesCache and eviction silently stopped working).
func TestEvictFromOrmWithFrameworkOrm(t *testing.T) {
	cache := frameworkorm.NewQueriesCache(
		map[string]contractsorm.Query{"mysql": nil, "tenant_1": nil, "tenant_2": nil},
		map[string]database.Config{"mysql": {}, "tenant_1": {Database: "t1"}, "tenant_2": {Database: "t2"}},
	)
	o := frameworkorm.NewOrm(context.Background(), nil, "mysql", database.Config{}, nil, cache, nil, nil, nil, nil)

	if !evictFromOrm(o, "tenant_1") {
		t.Fatal("evictFromOrm did not recognize the framework Orm layout")
	}
	if got := cachedConnectionNames(t, o); !reflect.DeepEqual(got, []string{"mysql", "tenant_2"}) {
		t.Fatalf("cache after evict = %v, want [mysql tenant_2]", got)
	}

	// Through the tenant-aware router, and for a name that is not cached.
	if !evictFromOrm(&ormRouter{base: o}, "tenant_2") || !evictFromOrm(o, "missing") {
		t.Fatal("evictFromOrm via router / missing name should succeed")
	}
	if got := cachedConnectionNames(t, o); !reflect.DeepEqual(got, []string{"mysql"}) {
		t.Fatalf("cache after second evict = %v, want [mysql]", got)
	}
}

func TestEvictFromOrmRejectsUnknownLayout(t *testing.T) {
	if evictFromOrm(&ormRouter{base: nil}, "x") {
		t.Fatal("nil base should not be reported as evicted")
	}
}
