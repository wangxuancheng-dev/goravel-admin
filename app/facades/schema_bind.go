package facades

import (
	"bytes"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/goravel/framework/contracts/database/schema"
)

// Goroutine-local tenant Schema.
//
// Migrations and seeders take no context: they call facades.Schema() / facades.Orm()
// directly, so the only way to point them at a tenant database without changing
// process-wide state is to pin a tenant Schema to the goroutine running them. The
// Schema and Orm routers consult this binding; the bound Schema's Orm doubles as the
// bound Orm, so inside a migration's transaction facades.Orm() uses that transaction.
//
// Everything else should route by ctx (OrmQuery(ctx)), which needs no binding.
var (
	tenantSchemaBinds sync.Map // goid(int64) -> schema.Schema
	activeBinds       atomic.Int64
)

// BindTenantSchema pins s for the current goroutine. Prefer WithBoundTenantSchema,
// which restores the previous binding; callers of this must UnbindTenantSchema.
func BindTenantSchema(s schema.Schema) {
	if s == nil {
		return
	}
	bindSchema(goroutineID(), s)
}

// UnbindTenantSchema clears the current goroutine's tenant Schema pin.
func UnbindTenantSchema() {
	unbindSchema(goroutineID())
}

// BoundTenantSchema returns the goroutine-local tenant Schema, if any. With nothing
// bound anywhere (the request path) it returns at once, without reading the stack.
func BoundTenantSchema() schema.Schema {
	if activeBinds.Load() == 0 {
		return nil
	}
	if v, ok := tenantSchemaBinds.Load(goroutineID()); ok {
		if s, ok := v.(schema.Schema); ok && s != nil {
			return s
		}
	}
	return nil
}

func bindSchema(gid int64, s schema.Schema) {
	if _, loaded := tenantSchemaBinds.Swap(gid, s); !loaded {
		activeBinds.Add(1)
	}
}

func unbindSchema(gid int64) {
	if _, loaded := tenantSchemaBinds.LoadAndDelete(gid); loaded {
		activeBinds.Add(-1)
	}
}

// TenantAwareSchemaInstalled reports whether MakeSchema returns the tenant-aware router.
func TenantAwareSchemaInstalled() bool {
	_, ok := Schema().(*schemaRouter)
	return ok
}

func goroutineID() int64 {
	var buf [64]byte
	n := runtimeStack(buf[:])
	// "goroutine 123 [running]..."
	b := buf[:n]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	i := bytes.IndexByte(b, ' ')
	if i <= 0 {
		return 0
	}
	id, err := strconv.ParseInt(string(b[:i]), 10, 64)
	if err != nil {
		return 0
	}
	return id
}

// runtimeStack is separated for tests; defaults to runtime.Stack.
var runtimeStack = func(buf []byte) int {
	return runtimeStackImpl(buf)
}
