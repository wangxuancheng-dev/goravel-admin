package facades

import (
	"bytes"
	"strconv"
	"sync"

	"github.com/goravel/framework/contracts/database/schema"
)

// Goroutine-local tenant Schema for parallel migrate.
// Migrations call migrations.Schema() which prefers this binding over the global facade.
var tenantSchemaBinds sync.Map // goid(int64) -> schema.Schema

// BindTenantSchema pins schema for the current goroutine (parallel tenant migrate).
// Caller must UnbindTenantSchema (usually via defer).
func BindTenantSchema(s schema.Schema) {
	if s == nil {
		return
	}
	tenantSchemaBinds.Store(goroutineID(), s)
}

// UnbindTenantSchema clears the current goroutine's tenant Schema pin.
func UnbindTenantSchema() {
	tenantSchemaBinds.Delete(goroutineID())
}

// BoundTenantSchema returns the goroutine-local tenant Schema, if any.
func BoundTenantSchema() schema.Schema {
	if v, ok := tenantSchemaBinds.Load(goroutineID()); ok {
		if s, ok := v.(schema.Schema); ok && s != nil {
			return s
		}
	}
	return nil
}

// TenantAwareSchemaInstalled reports whether MakeSchema returns the parallel-migrate router.
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
