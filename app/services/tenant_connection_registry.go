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

func forgetRegisteredLocked(connectionName string, freshORM bool) {
	if connectionName == "" {
		return
	}
	delete(registeredAt, connectionName)
	appfacades.EvictOrmConnectionCache(connectionName)
	if freshORM {
		if o := appfacades.Orm(); o != nil {
			o.Fresh()
		}
	}
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
