package facades

import (
	"context"
	"fmt"
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
//
// Also rebinds root Orm.Query and database.default for the duration: Schema.Create
// and some grammar paths otherwise still execute DDL on the platform DB while
// HasTable correctly inspects the tenant schema (goravel/framework Connection cache).
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
	if schema.GetConnection() == conn && schemaOrmDatabaseMatches(conn, schema) {
		return fn()
	}

	schemaConnMu.Lock()
	defer schemaConnMu.Unlock()

	if schema.GetConnection() == conn && schemaOrmDatabaseMatches(conn, schema) {
		return fn()
	}
	prev := schema.GetConnection()
	bindSchemaConnection(schema, conn)
	defer schema.SetConnection(prev)

	if !schemaOrmDatabaseMatches(conn, schema) {
		want := Config().GetString("database.connections."+conn+".database", "")
		return fmt.Errorf("tenant schema DatabaseName=%s want %s (orm connection cache)", schema.Orm().DatabaseName(), want)
	}

	rootOrm := Orm()
	if rootOrm != nil {
		prevQuery := rootOrm.Query()
		rootOrm.SetQuery(schema.Orm().Query())
		defer rootOrm.SetQuery(prevQuery)
	}

	prevDefault := Config().GetString("database.default")
	Config().Add("database.default", conn)
	defer Config().Add("database.default", prevDefault)

	return fn()
}

// bindSchemaConnection switches Schema to conn, rebuilding the Orm cache entry when
// DatabaseName still points at the platform DB (framework Connection cache bug).
func bindSchemaConnection(schema schema.Schema, conn string) {
	schema.SetConnection(conn)
	if schemaOrmDatabaseMatches(conn, schema) {
		return
	}
	EvictOrmConnectionCache(conn)
	schema.SetConnection(conn)
}

func schemaOrmDatabaseMatches(conn string, schema schema.Schema) bool {
	want := Config().GetString("database.connections."+conn+".database", "")
	if want == "" {
		return true
	}
	got := schema.Orm().DatabaseName()
	// Empty DatabaseName is not a match — force evict/rebind.
	if got == "" {
		return false
	}
	return got == want
}
