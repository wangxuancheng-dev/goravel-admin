package tenantcache

import (
	"strconv"
	"sync"
	"testing"
	"time"

	"gorm.io/gorm"

	"goravel/app/models"
)

type memStore struct {
	mu sync.Mutex
	m  map[string]any
}

func (s *memStore) Get(key string, def ...any) any {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v, ok := s.m[key]; ok {
		return v
	}
	if len(def) > 0 {
		return def[0]
	}
	return nil
}

func (s *memStore) GetString(key string, def ...string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v, ok := s.m[key].(string); ok {
		return v
	}
	if len(def) > 0 {
		return def[0]
	}
	return ""
}

func (s *memStore) Put(key string, value any, _ time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[key] = value
	return nil
}

func (s *memStore) Forget(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.m[key]
	delete(s.m, key)
	return ok
}

// Increment stores the counter as a string, like the Redis store.
func (s *memStore) Increment(key string, value ...int64) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	by := int64(1)
	if len(value) > 0 {
		by = value[0]
	}
	cur, _ := s.m[key].(string)
	n, _ := strconv.ParseInt(cur, 10, 64)
	n += by
	s.m[key] = strconv.FormatInt(n, 10)
	return n, nil
}

func useMemStore(t *testing.T) *memStore {
	t.Helper()
	s := &memStore{m: map[string]any{}}
	prev := cacheStore
	cacheStore = func() store { return s }
	l1.reset()
	prevHooked := hooksInstalled.Load()
	hooksInstalled.Store(true)
	t.Cleanup(func() {
		cacheStore = prev
		hooksInstalled.Store(prevHooked)
		l1.reset()
	})
	return s
}

func cacheRow(t *testing.T, s *memStore, ver string, row *models.Tenant) {
	t.Helper()
	raw, err := encode(row)
	if err != nil {
		t.Fatal(err)
	}
	_ = s.Put(rowKey(row.ID, ver), raw, time.Minute)
}

func tenantRow(id uint, code string) *models.Tenant {
	row := &models.Tenant{Code: code, Name: "cached", Status: models.TenantStatusActive}
	row.ID = id
	return row
}

// gob must keep the json:"-" fields: the connection and storage are built from them.
func TestEncodeKeepsSecretsAndSoftDelete(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	in := &models.Tenant{
		Code:          "acme",
		Password:      "enc:db-password",
		StorageSecret: "enc:v1:storage-secret",
		Maintenance:   true,
		LastOpAt:      &now,
	}
	in.ID = 7
	in.DeletedAt = gorm.DeletedAt{Time: now, Valid: true}

	raw, err := encode(in)
	if err != nil {
		t.Fatal(err)
	}
	out, err := decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	if out.ID != 7 || out.Code != "acme" || out.Password != in.Password || out.StorageSecret != in.StorageSecret ||
		!out.Maintenance || out.LastOpAt == nil || !out.LastOpAt.Equal(now) || !out.DeletedAt.Valid {
		t.Fatalf("round trip lost fields: %+v", out)
	}
}

func TestByIDServesCachedCopiesAndInvalidates(t *testing.T) {
	s := useMemStore(t)
	cacheRow(t, s, version(7), tenantRow(7, "acme"))

	a, err := ByID(7)
	if err != nil || a.Name != "cached" {
		t.Fatalf("cached read: %v %+v", err, a)
	}
	a.Name = "mutated by caller"
	b, _ := ByID(7)
	if b.Name != "cached" {
		t.Fatal("callers must get independent copies")
	}

	Invalidate(7)
	if IsCached(7) {
		t.Fatal("Invalidate must make the cached row unreachable")
	}
}

func TestInvalidateAllDropsEveryRow(t *testing.T) {
	s := useMemStore(t)
	cacheRow(t, s, version(7), tenantRow(7, "a"))
	cacheRow(t, s, version(8), tenantRow(8, "b"))
	if !IsCached(7) || !IsCached(8) {
		t.Fatal("precondition: both rows cached")
	}
	InvalidateAll()
	if IsCached(7) || IsCached(8) {
		t.Fatal("InvalidateAll must make every cached row unreachable")
	}
}

// The cache stays off until the invalidation hooks are installed.
func TestCacheDisabledWithoutHooks(t *testing.T) {
	useMemStore(t)
	hooksInstalled.Store(false)
	if ttl() != 0 {
		t.Fatal("without hooks the cache must be disabled")
	}
}

// A load that read the row before a write must not put the old row back: it stores
// under the version it read first, which the write has moved past.
func TestLoadRacingInvalidateDoesNotResurrectOldRow(t *testing.T) {
	s := useMemStore(t)
	for name, inval := range map[string]func(){"one": func() { Invalidate(7) }, "all": InvalidateAll} {
		verBeforeQuery := version(7)                         // loader reads the version, then queries the DB...
		inval()                                              // ...meanwhile the tenant is disabled and invalidated...
		cacheRow(t, s, verBeforeQuery, tenantRow(7, "acme")) // ...then the loader stores what it read.
		if IsCached(7) {
			t.Fatalf("%s: a stale load must not be visible after the invalidation", name)
		}
	}
}

func TestCounterReadsEveryStoreEncoding(t *testing.T) {
	s := useMemStore(t)
	n := int64(5)
	for _, v := range []any{"5", &n, int64(5), 5} {
		_ = s.Put(genKey(1), v, time.Minute)
		if got := counter(genKey(1)); got != "5" {
			t.Fatalf("counter(%T) = %q, want 5", v, got)
		}
	}
	_ = s.Forget(genKey(1))
	if got := counter(genKey(1)); got != "0" {
		t.Fatalf("missing counter = %q, want 0", got)
	}
}

func TestCorruptEntryIsDropped(t *testing.T) {
	s := useMemStore(t)
	_ = s.Put(rowKey(9, "0.0"), "not-base64-gob", time.Minute)
	if _, ok := get(rowKey(9, "0.0")); ok {
		t.Fatal("corrupt entry must be a miss")
	}
	if s.GetString(rowKey(9, "0.0")) != "" {
		t.Fatal("corrupt entry must be removed")
	}
}

func stmt(table string, model any) *gorm.DB {
	return &gorm.DB{Statement: &gorm.Statement{Table: table, Model: model}}
}

func TestWriteHookInvalidatesTheRightRows(t *testing.T) {
	s := useMemStore(t)
	reset := func() {
		cacheRow(t, s, version(7), tenantRow(7, "a"))
		cacheRow(t, s, version(8), tenantRow(8, "b"))
	}

	reset()
	afterWrite(stmt("tenants", tenantRow(7, "a"))) // Model(t).Update / Save(t) / Delete(t)
	if IsCached(7) || !IsCached(8) {
		t.Fatal("a write with the tenant model must invalidate only that tenant")
	}

	reset()
	afterWrite(stmt("tenants", &models.Tenant{})) // Where("id", x).Delete(&models.Tenant{})
	if IsCached(7) || IsCached(8) {
		t.Fatal("a bulk write must invalidate every tenant")
	}

	reset()
	afterWrite(stmt("tenant_domains", &models.TenantDomain{})) // another table
	failed := stmt("tenants", &models.Tenant{})
	failed.Error = gorm.ErrInvalidData
	afterWrite(failed) // failed write
	if !IsCached(7) || !IsCached(8) {
		t.Fatal("other tables and failed writes must not invalidate")
	}
}

func TestRawHookMatchesTenantsWritesOnly(t *testing.T) {
	s := useMemStore(t)
	for sql, want := range map[string]bool{
		"UPDATE tenants SET name = ? WHERE id = ?":          true,
		"update `tenants` set status = 0":                   true,
		"  DELETE FROM platform.tenants WHERE id = 3":       true,
		"INSERT INTO tenants (code) VALUES (?)":             true,
		"UPDATE tenant_domains SET status = 0":              false,
		"SELECT * FROM tenants":                             false,
		"DELETE FROM tenant_op_logs WHERE tenant_id = ?":    false,
		"UPDATE admins SET nickname = 'tenants' WHERE id=1": true, // false positive: extra invalidation only
	} {
		cacheRow(t, s, version(7), tenantRow(7, "a"))
		tx := &gorm.DB{Statement: &gorm.Statement{}}
		tx.Statement.SQL.WriteString(sql)
		afterRaw(tx)
		if got := !IsCached(7); got != want {
			t.Fatalf("%q: invalidated=%v, want %v", sql, got, want)
		}
	}
}

func TestTenantsTableMatchingIgnoresQuotingAndSchema(t *testing.T) {
	withTableName(t, "tenants")
	for name, want := range map[string]bool{
		"tenants": true, "`tenants`": true, `"public"."tenants"`: true, "platform.tenants": true,
		"tenants AS t": true, "TENANTS": true,
		"tenant_domains": false, "pre_tenants": false, "": false,
	} {
		if got := isTenantsTable(name); got != want {
			t.Fatalf("isTenantsTable(%q) = %v, want %v", name, got, want)
		}
	}
}

// With a table prefix the hooks must follow the real table name.
func TestHooksFollowTablePrefix(t *testing.T) {
	s := useMemStore(t)
	withTableName(t, "pre_tenants")

	cacheRow(t, s, version(7), tenantRow(7, "a"))
	afterWrite(stmt("tenants", &models.Tenant{})) // unprefixed name is another table now
	if !IsCached(7) {
		t.Fatal("unprefixed table must not match when a prefix is configured")
	}
	afterWrite(stmt("pre_tenants", &models.Tenant{}))
	if IsCached(7) {
		t.Fatal("the prefixed tenants table must invalidate")
	}

	cacheRow(t, s, version(7), tenantRow(7, "a"))
	tx := &gorm.DB{Statement: &gorm.Statement{}}
	tx.Statement.SQL.WriteString("UPDATE pre_tenants SET status = 0")
	afterRaw(tx)
	if IsCached(7) {
		t.Fatal("raw SQL on the prefixed table must invalidate")
	}
}

func TestResetHooksDisablesAndDropsCache(t *testing.T) {
	s := useMemStore(t)
	cacheRow(t, s, version(7), tenantRow(7, "a"))
	ResetHooks()
	if ttl() != 0 {
		t.Fatal("cache must be off until hooks are installed again")
	}
	hooksInstalled.Store(true)
	if IsCached(7) {
		t.Fatal("rows cached before the reset must not be served afterwards")
	}
}

func withTableName(t *testing.T, name string) {
	t.Helper()
	prev, _ := tableName.Load().(string)
	setTable(name)
	t.Cleanup(func() { setTable(prev) })
}

// countingStore counts reads that reach the shared store.
type countingStore struct {
	*memStore
	reads int
}

func (c *countingStore) Get(key string, def ...any) any {
	c.reads++
	return c.memStore.Get(key, def...)
}

func (c *countingStore) GetString(key string, def ...string) string {
	c.reads++
	return c.memStore.GetString(key, def...)
}

// After the first read, a tenant lookup reads only its two version counters from the
// shared store; the row (and the code mapping) come from the in-process copy.
func TestLocalCopyCutsSharedStoreReads(t *testing.T) {
	s := useMemStore(t)
	cs := &countingStore{memStore: s}
	cacheStore = func() store { return cs }
	cacheRow(t, s, version(7), tenantRow(7, "acme"))
	_ = s.Put(codeKey("acme"), "7", time.Minute)

	_, _ = ByCode("acme") // warm: code mapping + versions + row from the shared store
	cs.reads = 0
	got, err := ByCode("acme")
	if err != nil || got.ID != 7 {
		t.Fatalf("ByCode: %v %+v", err, got)
	}
	if cs.reads != 2 {
		t.Fatalf("warm ByCode read the shared store %d times, want 2 (the version counters)", cs.reads)
	}
	cs.reads = 0
	_, _ = ByID(7)
	if cs.reads != 2 {
		t.Fatalf("warm ByID read the shared store %d times, want 2", cs.reads)
	}

	// An invalidation on any node moves the version, so the local copy is not used.
	Invalidate(7)
	disabled := tenantRow(7, "acme")
	disabled.Status = models.TenantStatusDisabled
	cacheRow(t, s, version(7), disabled)
	got, err = ByID(7)
	if err != nil || got.Status != models.TenantStatusDisabled {
		t.Fatalf("after Invalidate the local copy must not be served: %v %+v", err, got)
	}
}
