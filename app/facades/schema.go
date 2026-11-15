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
//
// With the tenant-aware routers installed (always, via DatabaseServiceProvider), an
// independent tenant Schema+Orm is bound to the current goroutine only. Nothing
// process-wide changes, so concurrent requests can never be routed to this tenant.
// Already bound to the same connection (e.g. inside WithTenantConnection) → fn runs as is.
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
	if bound := BoundTenantSchema(); bound != nil && bound.GetConnection() == conn {
		return fn()
	}
	if !TenantRoutersInstalled() {
		return withGlobalSchemaContext(conn, fn)
	}
	sch, err := TenantSchemaFor(conn)
	if err != nil {
		return err
	}
	return WithBoundTenantSchema(sch, fn)
}

// TenantSchemaFor returns an independent Schema on conn, verified to be connected
// to the configured tenant database (evicts a stale Orm cache entry and retries once).
func TenantSchemaFor(conn string) (schema.Schema, error) {
	sch, err := tenantSchemaConnection(conn)
	if err != nil {
		return nil, err
	}
	if schemaOrmDatabaseMatches(conn, sch) {
		return sch, nil
	}
	EvictOrmConnectionCache(conn)
	if sch, err = tenantSchemaConnection(conn); err != nil {
		return nil, err
	}
	if !schemaOrmDatabaseMatches(conn, sch) {
		want := Config().GetString("database.connections."+conn+".database", "")
		return nil, fmt.Errorf("tenant schema session DATABASE()=%s DatabaseName=%s want %s",
			schemaSessionDatabase(sch), sch.Orm().DatabaseName(), want)
	}
	return sch, nil
}

func tenantSchemaConnection(conn string) (sch schema.Schema, err error) {
	// Schema.Connection log.Panics when the connection is missing.
	defer func() {
		if r := recover(); r != nil {
			sch, err = nil, fmt.Errorf("tenant schema connection %s: %v", conn, r)
		}
	}()
	sch = Schema().Connection(conn)
	if sch == nil || sch.Orm() == nil || sch.Orm().Query() == nil {
		return nil, fmt.Errorf("tenant schema connection %s unavailable", conn)
	}
	return sch, nil
}

// withGlobalSchemaContext is the legacy path (routers not installed): it flips the
// global Schema, root Orm query and database.default, serialized by schemaConnMu.
// Concurrent unbound queries can observe the flip, so it is only a fallback.
func withGlobalSchemaContext(conn string, fn func() error) error {
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
		return fmt.Errorf("tenant schema session DATABASE()=%s DatabaseName=%s want %s",
			schemaSessionDatabase(schema), schema.Orm().DatabaseName(), want)
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

// bindSchemaConnection switches Schema to conn. Always evicts the Orm connection
// cache first: a cache hit can report the tenant DatabaseName while Query still
// executes against the platform default database.
func bindSchemaConnection(schema schema.Schema, conn string) {
	EvictOrmConnectionCache(conn)
	schema.SetConnection(conn)
	if schemaOrmDatabaseMatches(conn, schema) {
		return
	}
	// Second pass after another writer may have repopulated a stale cache entry.
	EvictOrmConnectionCache(conn)
	schema.SetConnection(conn)
}

func schemaSessionDatabase(schema schema.Schema) string {
	if schema == nil || schema.Orm() == nil || schema.Orm().Query() == nil {
		return ""
	}
	var name string
	if err := schema.Orm().Query().Raw("SELECT DATABASE()").Scan(&name); err != nil {
		return ""
	}
	return name
}

func schemaOrmDatabaseMatches(conn string, schema schema.Schema) bool {
	want := Config().GetString("database.connections."+conn+".database", "")
	if want == "" {
		return true
	}
	// Live session DB is authoritative; DatabaseName() can lie on cache hits.
	if got := schemaSessionDatabase(schema); got != "" {
		return got == want
	}
	got := schema.Orm().DatabaseName()
	if got == "" {
		return false
	}
	return got == want
}
