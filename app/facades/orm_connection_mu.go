package facades

import (
	"sync"

	"github.com/goravel/framework/contracts/database/orm"
)

// ormConnectionMu serializes tenant connection config writes and Orm/Schema
// Connection builds. Viper and Goravel MySQL config maps are not concurrent-safe.
var ormConnectionMu sync.Mutex

// LockOrmConnectionBuild locks the shared connection-build / config-write mutex.
// Callers that Config.Add database.connections.* must hold this around the write
// and any immediate Connection() that follows.
func LockOrmConnectionBuild() { ormConnectionMu.Lock() }

// UnlockOrmConnectionBuild unlocks LockOrmConnectionBuild.
func UnlockOrmConnectionBuild() { ormConnectionMu.Unlock() }

// BuildOrmConnection opens (or returns cached) an Orm for name under the
// connection-build mutex. Prefer this over Orm().Connection from paths that
// may run parallel with migrate/seed (e.g. warmConnection).
func BuildOrmConnection(name string) orm.Orm {
	ormConnectionMu.Lock()
	defer ormConnectionMu.Unlock()
	return buildOrmConnectionLocked(name)
}

// BuildOrmConnectionLocked is BuildOrmConnection when the caller already holds
// LockOrmConnectionBuild.
func BuildOrmConnectionLocked(name string) orm.Orm {
	return buildOrmConnectionLocked(name)
}

func buildOrmConnectionLocked(name string) orm.Orm {
	o := Orm()
	if o == nil {
		return nil
	}
	if r, ok := o.(*ormRouter); ok {
		return r.base.Connection(name)
	}
	return o.Connection(name)
}