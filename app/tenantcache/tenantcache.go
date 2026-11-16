// Package tenantcache caches platform tenants rows, which every tenant request reads
// (tenant resolution, connection and storage setup).
//
// Rows live in the shared cache store (facades.Cache: Redis in production), so an
// invalidation on one node takes effect on every node at once: disabling a tenant or
// putting it in maintenance is not delayed by the cache.
//
// Invalidation is automatic (InstallHooks): every successful write to the tenants
// table through the platform connection's gorm — Model(t).Update, Save, Delete, bulk
// Where(...).Delete, even Exec("UPDATE tenants ...") — drops the affected row, or
// every row when the write does not identify one. Only SQL run outside the app (a
// mysql client) waits for the TTL.
//
// Each row key carries the tenant's generation and a global generation; invalidation
// bumps one of them, and a loader stores the row under the generations it read
// *before* querying the database. A load racing an invalidation therefore writes to a
// key nobody reads any more, instead of putting the pre-write row back.
//
// Processes only share invalidations through a shared store: with CACHE_STORE=memory,
// run a single process or rows may be stale for up to the TTL in the others.
package tenantcache

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"errors"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/goravel/framework/facades"
	"github.com/spf13/cast"
	"golang.org/x/sync/singleflight"

	appfacades "goravel/app/facades"
	"goravel/app/models"
)

// ErrNotFound means no (non-deleted) tenant matches.
var ErrNotFound = errors.New("tenantcache: tenant not found")

const (
	keyPrefix = "tenant_row:v3:"
	allGenKey = keyPrefix + "gen:all"
)

var group singleflight.Group

func genKey(id uint) string { return keyPrefix + "gen:" + strconv.FormatUint(uint64(id), 10) }
func rowKey(id uint, version string) string {
	return keyPrefix + "id:" + strconv.FormatUint(uint64(id), 10) + ":v" + version
}
func codeKey(code string) string { return keyPrefix + "code:" + code }

// ttl is tenancy.tenant_cache_ttl seconds; 0 (or no InstallHooks yet) disables the cache.
func ttl() time.Duration {
	if !hooksInstalled.Load() {
		return 0
	}
	sec := facades.Config().GetInt("tenancy.tenant_cache_ttl", 60)
	if sec <= 0 {
		return 0
	}
	return time.Duration(sec) * time.Second
}

// counter reads a generation counter ("0" before the first bump).
func counter(key string) string {
	switch v := cacheStore().Get(key).(type) {
	case nil:
		return "0"
	case *int64:
		return strconv.FormatInt(atomic.LoadInt64(v), 10)
	case *atomic.Int64:
		return strconv.FormatInt(v.Load(), 10)
	case *atomic.Int32:
		return strconv.FormatInt(int64(v.Load()), 10)
	default:
		if s := cast.ToString(v); s != "" {
			return s
		}
		return "0"
	}
}

// version is the tenant's generation plus the global one: bumping either makes every
// row stored under the previous version unreachable.
func version(id uint) string { return counter(genKey(id)) + "." + counter(allGenKey) }

// ByID returns a copy of the non-deleted tenant with id (callers may mutate it).
func ByID(id uint) (*models.Tenant, error) {
	if id == 0 {
		return nil, ErrNotFound
	}
	d := ttl()
	if d <= 0 {
		return loadByID(id, "", 0)
	}
	ver := version(id)
	key := rowKey(id, ver)
	if t, ok := l1.row(key); ok {
		return t, nil
	}
	if t, ok := get(key); ok {
		l1.putRow(key, t, d)
		return t, nil
	}
	v, err, _ := group.Do(key, func() (any, error) {
		t, err := loadByID(id, ver, d)
		if err == nil {
			l1.putRow(key, t, d)
		}
		return t, err
	})
	if err != nil {
		return nil, err
	}
	return clone(v.(*models.Tenant)), nil
}

// ByCode returns a copy of the non-deleted tenant with code (lower-cased, trimmed).
func ByCode(code string) (*models.Tenant, error) {
	code = strings.ToLower(strings.TrimSpace(code))
	if code == "" {
		return nil, ErrNotFound
	}
	d := ttl()
	if d > 0 {
		// code -> id is cached separately; the row itself goes through ByID, so its
		// version check also covers lookups by code.
		id, ok := l1.code(code)
		if !ok {
			if n, err := strconv.ParseUint(cacheStore().GetString(codeKey(code), ""), 10, 64); err == nil && n > 0 {
				id, ok = uint(n), true
			}
		}
		if ok {
			if t, err := ByID(id); err == nil && t.Code == code {
				l1.putCode(code, id, d)
				return t, nil
			}
			// Mapping points at a deleted or re-coded tenant.
			l1.forgetCode(code)
			_ = cacheStore().Forget(codeKey(code))
		}
	}
	v, err, _ := group.Do(codeKey(code), func() (any, error) {
		var row struct{ ID uint }
		if err := appfacades.PlatformQuery().Model(&models.Tenant{}).Select("id").Where("code", code).First(&row); err != nil {
			return nil, err
		}
		if row.ID == 0 {
			return nil, ErrNotFound
		}
		if d > 0 {
			_ = cacheStore().Put(codeKey(code), strconv.FormatUint(uint64(row.ID), 10), d)
		}
		return row.ID, nil
	})
	if err != nil {
		return nil, err
	}
	t, err := ByID(v.(uint))
	if err != nil {
		return nil, err
	}
	if t.Code != code {
		return nil, ErrNotFound
	}
	return t, nil
}

// Invalidate makes every node stop serving the cached row of tenant id. Writes
// through the ORM call it automatically (InstallHooks).
func Invalidate(id uint) {
	if id == 0 {
		return
	}
	if _, err := cacheStore().Increment(genKey(id)); err != nil {
		logInvalidateFailure("tenant "+strconv.FormatUint(uint64(id), 10), err)
		// Without a generation bump, at least drop the current row.
		_ = cacheStore().Forget(rowKey(id, version(id)))
	}
}

// InvalidateAll makes every node stop serving any cached tenant row (a write that
// does not identify the rows it changed).
func InvalidateAll() {
	if _, err := cacheStore().Increment(allGenKey); err != nil {
		logInvalidateFailure("all tenants", err)
	}
}

// logInvalidateFailure reports an invalidation the cache store rejected: rows cached
// before it may be served until the TTL.
func logInvalidateFailure(what string, err error) {
	if l := facades.Log(); l != nil {
		l.Errorf("tenantcache: invalidating %s failed, cached rows may be stale for up to the TTL: %v", what, err)
	}
}

// IsCached reports whether tenant id's current row is in the cache (tests, metrics).
func IsCached(id uint) bool {
	return cacheStore().GetString(rowKey(id, version(id)), "") != ""
}

// loadByID queries the row and, when d > 0, stores it under ver — the version read
// before the query, so a concurrent invalidation makes this write unreachable.
func loadByID(id uint, ver string, d time.Duration) (*models.Tenant, error) {
	var t models.Tenant
	if err := appfacades.PlatformQuery().Where("id", id).First(&t); err != nil {
		return nil, err
	}
	if t.ID == 0 {
		return nil, ErrNotFound
	}
	if d > 0 {
		if enc, err := encode(&t); err == nil {
			_ = cacheStore().Put(rowKey(id, ver), enc, d)
		}
	}
	return &t, nil
}

func get(key string) (*models.Tenant, bool) {
	raw := cacheStore().GetString(key, "")
	if raw == "" {
		return nil, false
	}
	t, err := decode(raw)
	if err != nil {
		_ = cacheStore().Forget(key)
		return nil, false
	}
	return t, true
}

// encode uses gob, not JSON: Password and StorageSecret are `json:"-"` (they hold
// APP_KEY-sealed ciphertext, same as in the tenants table).
func encode(t *models.Tenant) (string, error) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(t); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

func decode(raw string) (*models.Tenant, error) {
	b, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, err
	}
	var t models.Tenant
	if err := gob.NewDecoder(bytes.NewReader(b)).Decode(&t); err != nil {
		return nil, err
	}
	return &t, nil
}

// clone gives each caller its own copy of a row shared by a singleflight call.
func clone(t *models.Tenant) *models.Tenant {
	c := *t
	return &c
}

// cacheStore is the shared cache store (swapped in unit tests).
var cacheStore = func() store { return facades.Cache() }

type store interface {
	Get(key string, def ...any) any
	GetString(key string, def ...string) string
	Put(key string, value any, t time.Duration) error
	Forget(key string) bool
	Increment(key string, value ...int64) (int64, error)
}
