package services

import (
	"sync"
	"time"

	"github.com/goravel/framework/facades"

	appfacades "goravel/app/facades"
)

// Process-local tenant connection registry with optional idle TTL and max size.
// Small fleets: defaults leave behavior close to "keep forever".
// Large fleets: set TENANCY_REGISTERED_MAX / TENANCY_REGISTERED_IDLE_TTL (or rely on idle TTL default).
var (
	registeredMu   sync.Mutex
	registeredAt   = map[string]time.Time{} // connection_name -> lastUsed
)

func registeredMax() int {
	return facades.Config().GetInt("tenancy.registered_max", 0)
}

func registeredIdleTTL() time.Duration {
	sec := facades.Config().GetInt("tenancy.registered_idle_ttl", 900)
	if sec <= 0 {
		return 0
	}
	return time.Duration(sec) * time.Second
}

// RegisteredConnCount returns how many tenant pools are registered in this process.
func RegisteredConnCount() int {
	registeredMu.Lock()
	defer registeredMu.Unlock()
	return len(registeredAt)
}

// ResetRegisteredConnsForTest clears the in-process registry (unit tests only).
func ResetRegisteredConnsForTest() {
	registeredMu.Lock()
	defer registeredMu.Unlock()
	registeredAt = map[string]time.Time{}
}

func touchRegisteredLocked(connectionName string) {
	if connectionName == "" {
		return
	}
	registeredAt[connectionName] = time.Now()
}

func isRegisteredLocked(connectionName string) bool {
	_, ok := registeredAt[connectionName]
	return ok
}

func markRegisteredLocked(connectionName string) {
	touchRegisteredLocked(connectionName)
}

// forgetRegisteredLocked drops registry state for connectionName.
// teardown=true closes the pool and clears config (Forget / DropStorage).
// teardown=false only evicts the Orm query cache (idle/max eviction) so a later
// EnsureRegistered can rebuild without a process-wide Orm.Fresh().
func forgetRegisteredLocked(connectionName string, teardown bool) {
	if connectionName == "" {
		return
	}
	delete(registeredAt, connectionName)
	if teardown {
		teardownTenantConnectionLocked(connectionName)
		return
	}
	appfacades.EvictOrmConnectionCache(connectionName)
}

// teardownTenantConnectionLocked closes the pool and clears config for name.
// Does not take registeredMu. Avoids Orm.Fresh() which breaks later feature
// tests when DropStorage runs mid-suite.
func teardownTenantConnectionLocked(connectionName string) {
	appfacades.LockOrmConnectionBuild()
	defer appfacades.UnlockOrmConnectionBuild()

	// Close while DSN still present (DropStorage calls Forget before DROP DATABASE).
	if facades.Config().GetString("database.connections."+connectionName+".database", "") != "" {
		if to := appfacades.BuildOrmConnectionLocked(connectionName); to != nil {
			if db, err := to.DB(); err == nil && db != nil {
				_ = db.Close()
			}
		}
	}
	appfacades.EvictOrmConnectionCacheLocked(connectionName)
	// Empty map (not nil): prevents driver init with a stale DSN after drop.
	facades.Config().Add("database.connections."+connectionName, map[string]any{})
	facades.Config().Add("database.connections."+connectionName+"_maint", map[string]any{})
}

// evictRegisteredLocked drops idle and/or oldest entries so a new registration can proceed.
// protect is never evicted. Returns whether any entry was removed.
func evictRegisteredLocked(protect string) (evicted bool) {
	now := time.Now()
	if ttl := registeredIdleTTL(); ttl > 0 {
		for name, last := range registeredAt {
			if name == protect {
				continue
			}
			if now.Sub(last) >= ttl {
				forgetRegisteredLocked(name, false)
				evicted = true
			}
		}
	}

	max := registeredMax()
	if max <= 0 {
		return evicted
	}
	for len(registeredAt) >= max {
		oldestName := ""
		var oldestTime time.Time
		for name, last := range registeredAt {
			if name == protect {
				continue
			}
			if oldestName == "" || last.Before(oldestTime) {
				oldestName = name
				oldestTime = last
			}
		}
		if oldestName == "" {
			break
		}
		forgetRegisteredLocked(oldestName, false)
		evicted = true
	}
	return evicted
}

func freshORMIfNeeded(evicted bool) {
	if !evicted {
		return
	}
	if o := appfacades.Orm(); o != nil {
		o.Fresh()
	}
}
