// Package tenantcache caches platform tenants rows, which every tenant request reads
// (tenant resolution, connection and storage setup).
//
// Rows live in the shared cache store (facades.Cache: Redis in production), so an
// invalidation on one node takes effect on every node at once: disabling a tenant or
// putting it in maintenance is not delayed by the cache. Every ORM write of a tenant row
// invalidates it (models.TenantChanged); the TTL only bounds writes made outside the app.
//
// Invalidation bumps a per-tenant generation that is part of the row key, and a loader
// stores the row under the generation it read *before* querying the database. A load
// that races with an invalidation therefore writes to a key nobody reads any more,
// instead of putting the pre-write row back into the cache.
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

const keyPrefix = "tenant_row:v2:"

var group singleflight.Group

func init() {
	models.TenantChanged = Invalidate
}

func genKey(id uint) string { return keyPrefix + "gen:" + strconv.FormatUint(uint64(id), 10) }
func rowKey(id uint, gen string) string {
	return keyPrefix + "id:" + strconv.FormatUint(uint64(id), 10) + ":g" + gen
}
func codeKey(code string) string { return keyPrefix + "code:" + code }

// ttl is tenancy.tenant_cache_ttl seconds; 0 disables the cache.
func ttl() time.Duration {
	sec := facades.Config().GetInt("tenancy.tenant_cache_ttl", 60)
	if sec <= 0 {
		return 0
	}
	return time.Duration(sec) * time.Second
}

// generation returns the current generation of tenant id ("0" before any write).
func generation(id uint) string {
	switch v := cacheStore().Get(genKey(id)).(type) {
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

// ByID returns a copy of the non-deleted tenant with id (callers may mutate it).
func ByID(id uint) (*models.Tenant, error) {
	if id == 0 {
		return nil, ErrNotFound
	}
	d := ttl()
	if d <= 0 {
		return loadByID(id, "", 0)
	}
	gen := generation(id)
	if t, ok := get(rowKey(id, gen)); ok {
		return t, nil
	}
	v, err, _ := group.Do(rowKey(id, gen), func() (any, error) {
		return loadByID(id, gen, d)
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
		// generation check also covers lookups by code.
		if id, err := strconv.ParseUint(cacheStore().GetString(codeKey(code), ""), 10, 64); err == nil && id > 0 {
			if t, err := ByID(uint(id)); err == nil && t.Code == code {
				return t, nil
			}
			// Mapping points at a deleted or re-coded tenant.
			_ = cacheStore().Forget(codeKey(code))
		}
	}
	v, err, _ := group.Do(codeKey(code), func() (any, error) {
		var row struct{ ID uint }
		if err := appfacades.PlatformOrmQuery(nil).Model(&models.Tenant{}).Select("id").Where("code", code).First(&row); err != nil {
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

// Invalidate makes every node stop serving the cached row of tenant id.
func Invalidate(id uint) {
	if id == 0 {
		return
	}
	if _, err := cacheStore().Increment(genKey(id)); err != nil {
		// Without a generation bump, at least drop the current row.
		_ = cacheStore().Forget(rowKey(id, generation(id)))
	}
}

// loadByID queries the row and, when d > 0, stores it under gen — the generation read
// before the query, so a concurrent Invalidate makes this write unreachable.
func loadByID(id uint, gen string, d time.Duration) (*models.Tenant, error) {
	var t models.Tenant
	if err := appfacades.PlatformOrmQuery(nil).Where("id", id).First(&t); err != nil {
		return nil, err
	}
	if t.ID == 0 {
		return nil, ErrNotFound
	}
	if d > 0 {
		if enc, err := encode(&t); err == nil {
			_ = cacheStore().Put(rowKey(id, gen), enc, d)
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
