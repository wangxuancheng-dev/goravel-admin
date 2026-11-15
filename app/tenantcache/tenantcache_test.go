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
	n, _ := strconv.ParseInt(stringOf(s.m[key]), 10, 64)
	n += by
	s.m[key] = strconv.FormatInt(n, 10)
	return n, nil
}

func stringOf(v any) string {
	s, _ := v.(string)
	return s
}

func useMemStore(t *testing.T) *memStore {
	t.Helper()
	s := &memStore{m: map[string]any{}}
	prev := cacheStore
	cacheStore = func() store { return s }
	t.Cleanup(func() { cacheStore = prev })
	return s
}

func putRow(t *testing.T, s *memStore, gen string, row *models.Tenant) {
	t.Helper()
	raw, err := encode(row)
	if err != nil {
		t.Fatal(err)
	}
	_ = s.Put(rowKey(row.ID, gen), raw, time.Minute)
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
	row := &models.Tenant{Code: "acme", Name: "cached"}
	row.ID = 7
	putRow(t, s, generation(7), row)

	a, err := ByID(7)
	if err != nil || a.Name != "cached" {
		t.Fatalf("cached read: %v %+v", err, a)
	}
	a.Name = "mutated by caller"
	b, _ := ByID(7)
	if b.Name != "cached" {
		t.Fatal("callers must get independent copies")
	}

	models.TenantChanged(7) // what an ORM write of the row triggers
	if _, ok := get(rowKey(7, generation(7))); ok {
		t.Fatal("TenantChanged must make the cached row unreachable")
	}
}

// A load that read the row before a write must not put the old row back: it stores
// under the generation it read first, which the write has moved past.
func TestLoadRacingInvalidateDoesNotResurrectOldRow(t *testing.T) {
	s := useMemStore(t)
	genBeforeQuery := generation(7) // loader reads the generation, then queries the DB...

	models.TenantChanged(7) // ...meanwhile the tenant is disabled and invalidated...

	stale := &models.Tenant{Code: "acme", Status: models.TenantStatusActive}
	stale.ID = 7
	putRow(t, s, genBeforeQuery, stale) // ...then the loader stores what it read.

	if _, ok := get(rowKey(7, generation(7))); ok {
		t.Fatal("a stale load must not be visible after the invalidation")
	}
}

func TestGenerationReadsEveryStoreEncoding(t *testing.T) {
	s := useMemStore(t)
	n := int64(5)
	for _, v := range []any{"5", &n, int64(5), 5} {
		_ = s.Put(genKey(1), v, time.Minute)
		if got := generation(1); got != "5" {
			t.Fatalf("generation(%T) = %q, want 5", v, got)
		}
	}
	_ = s.Forget(genKey(1))
	if got := generation(1); got != "0" {
		t.Fatalf("missing generation = %q, want 0", got)
	}
}

func TestCorruptEntryIsDropped(t *testing.T) {
	s := useMemStore(t)
	_ = s.Put(rowKey(9, "0"), "not-base64-gob", time.Minute)
	if _, ok := get(rowKey(9, "0")); ok {
		t.Fatal("corrupt entry must be a miss")
	}
	if s.GetString(rowKey(9, "0")) != "" {
		t.Fatal("corrupt entry must be removed")
	}
}
