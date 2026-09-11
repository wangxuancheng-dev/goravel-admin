package facades

import (
	"context"

	"github.com/goravel/framework/contracts/database/schema"

	"goravel/app/tenancy"
	"goravel/app/tenancyctx"
)

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
// When tenancy is off or ctx has no tenant connection, fn runs against the current Schema connection
// (e.g. already switched by WithTenantConnection).
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
	prev := schema.GetConnection()
	schema.SetConnection(conn)
	defer schema.SetConnection(prev)
	return fn()
}
