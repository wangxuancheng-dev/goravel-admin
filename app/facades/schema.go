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
	schema := Schema()
	if !tenancy.Enabled() || ctx == nil {
		return schema.HasTable(table)
	}
	conn, ok := tenancyctx.ConnectionFrom(ctx)
	if !ok || conn == "" {
		return schema.HasTable(table)
	}
	prev := schema.GetConnection()
	schema.SetConnection(conn)
	defer schema.SetConnection(prev)
	return schema.HasTable(table)
}

// SchemaHasColumn checks HasColumn on the tenant connection when ctx is bound.
func SchemaHasColumn(ctx context.Context, table, column string) bool {
	schema := Schema()
	if !tenancy.Enabled() || ctx == nil {
		return schema.HasColumn(table, column)
	}
	conn, ok := tenancyctx.ConnectionFrom(ctx)
	if !ok || conn == "" {
		return schema.HasColumn(table, column)
	}
	prev := schema.GetConnection()
	schema.SetConnection(conn)
	defer schema.SetConnection(prev)
	return schema.HasColumn(table, column)
}

// SchemaConnectionKey returns the active schema connection name (for caches).
func SchemaConnectionKey() string {
	schema := Schema()
	if conn := schema.GetConnection(); conn != "" {
		return conn
	}
	return Config().GetString("database.default", "mysql")
}
