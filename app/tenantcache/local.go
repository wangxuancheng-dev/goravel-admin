package tenantcache

import (
	"sync"
	"time"

	"goravel/app/models"
)

// local is an in-process copy of what the shared store holds, so a tenant request
// reads only the two version counters from the shared store instead of also the row
// (and, by code, the code -> id mapping).
//
// It cannot serve stale data: rows are keyed by the versioned row key, which changes
// on every invalidation, and code -> id mappings are only a hint that ByCode checks
// against the row it leads to. Entries expire with the TTL; the map is dropped whole
// when it reaches localMax entries.
type local struct {
	mu    sync.Mutex
	rows  map[string]localRow // rowKey -> decoded row
	codes map[string]localID  // code -> id
}

type localRow struct {
	t   *models.Tenant
	exp time.Time
}

type localID struct {
	id  uint
	exp time.Time
}

const localMax = 10000

var l1 = &local{rows: map[string]localRow{}, codes: map[string]localID{}}

func (c *local) row(key string) (*models.Tenant, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.rows[key]
	if !ok || time.Now().After(e.exp) {
		return nil, false
	}
	return clone(e.t), true
}

func (c *local) putRow(key string, t *models.Tenant, d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.rows) >= localMax {
		c.rows = map[string]localRow{}
	}
	c.rows[key] = localRow{t: clone(t), exp: time.Now().Add(d)}
}

func (c *local) code(code string) (uint, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.codes[code]
	if !ok || time.Now().After(e.exp) {
		return 0, false
	}
	return e.id, true
}

func (c *local) putCode(code string, id uint, d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.codes) >= localMax {
		c.codes = map[string]localID{}
	}
	c.codes[code] = localID{id: id, exp: time.Now().Add(d)}
}

func (c *local) forgetCode(code string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.codes, code)
}

func (c *local) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rows = map[string]localRow{}
	c.codes = map[string]localID{}
}
