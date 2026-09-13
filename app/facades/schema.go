package facades

import (
	"context"
	"sync"

	"github.com/goravel/framework/contracts/database/schema"

	"goravel/app/tenancy"
	"goravel/app/tenancyctx"
)

// schemaConnMu serializes Schema.SetConnection for request-path DDL/HasTable and
// TenantConnectionService.WithTenantConnection (via SchemaConnLock).
var schemaConnMu sync.Mutex

// SchemaConnLock exposes the schema connection mutex for tenant migrate/seed paths.
func SchemaConnLock() *sync.Mutex {
	return &schemaConnMu
}

func Schema() schema.Schema {
	return App().MakeSchema()
}

// SchemaHasTable checks HasTable on the tenant connection when ctx is bound.
func SchemaHasTable(ctx context.Context, table string) bool {
	var exists bool
	_ = WithSchemaContext(ctx, func() error {
		exists = Schema().HasTable(table)
		return nil
	})
	return exists
}

// SchemaHasColumn checks HasColumn on the tenant connection when ctx is bound.
func SchemaHasColumn(ctx context.Context, table, column string) bool {
	var exists bool
	_ = WithSchemaContext(ctx, func() error {
		exists = Schema().HasColumn(table, column)
		return nil
	})
	return exists
}

// SchemaConnectionKey returns the active schema connection name (for caches).
func SchemaConnectionKey() string {
	schema := Schema()
	if conn := schema.GetConnection(); conn != "" {
		return conn
	}
	return Config().GetString("database.default", "mysql")
}

// SchemaConnectionKeyFrom prefers tenant connection from ctx, else current Schema connection.
func SchemaConnectionKeyFrom(ctx context.Context) string {
	if tenancy.Enabled() {
		if conn, ok := tenancyctx.ConnectionFrom(ctx); ok && conn != "" {
			return conn
		}
	}
	return SchemaConnectionKey()
}

// WithSchemaContext runs fn with Schema (and DDL) bound to the tenant connection from ctx.
// Concurrent callers are serialized. If Schema is already on the target connection
// (e.g. inside WithTenantConnection), fn runs without re-locking to avoid deadlock.
func WithSchemaContext(ctx context.Context, fn func() error) error {
	if fn == nil {
		return nil
	}
	if !tenancy.Enabled() || ctx == nil {
		return fn()
	}
	conn, ok := tenancyctx.ConnectionFrom(ctx)
	if !ok || conn == "" {
		return fn()
	}
	schema := Schema()
	if schema.GetConnection() == conn {
		return fn()
	}

	schemaConnMu.Lock()
	defer schemaConnMu.Unlock()

	if schema.GetConnection() == conn {
		return fn()
	}
	prev := schema.GetConnection()
	// Evict so SetConnection rebuilds Orm with tenant dbConfig (see EvictOrmConnectionCache).
	EvictOrmConnectionCache(conn)
	schema.SetConnection(conn)
	defer schema.SetConnection(prev)
	return fn()
}
