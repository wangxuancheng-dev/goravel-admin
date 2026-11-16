package facades

import (
	"context"
	"fmt"

	"github.com/goravel/framework/contracts/database/schema"

	"goravel/app/tenancy"
	"goravel/app/tenancyctx"
	"goravel/app/tenantdb"
)

func Schema() schema.Schema {
	return App().MakeSchema()
}

// SchemaHasTable checks HasTable on the tenant connection when ctx is bound.
func SchemaHasTable(ctx context.Context, table string) bool {
	s, err := schemaFor(ctx)
	if err != nil {
		return false
	}
	return s.HasTable(table)
}

// SchemaHasColumn checks HasColumn on the tenant connection when ctx is bound.
func SchemaHasColumn(ctx context.Context, table, column string) bool {
	s, err := schemaFor(ctx)
	if err != nil {
		return false
	}
	return s.HasColumn(table, column)
}

// SchemaTables lists the table names of ctx's connection (the tenant's when ctx is
// bound), through the framework Schema.
func SchemaTables(ctx context.Context) ([]string, error) {
	s, err := schemaFor(ctx)
	if err != nil {
		return nil, err
	}
	tables, err := s.GetTables()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(tables))
	for _, t := range tables {
		names = append(names, t.Name)
	}
	return names, nil
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

// WithSchemaContext runs fn with facades.Schema() / facades.Orm() pointed at the tenant
// connection from ctx (for code that cannot take a Schema, e.g. migration-style table
// creators). A fresh tenant Schema is bound to the current goroutine only; nothing
// process-wide changes, so concurrent requests are never routed to this tenant.
// Already bound to the same connection (e.g. a migrate / seed goroutine) → fn runs as is.
func WithSchemaContext(ctx context.Context, fn func() error) error {
	if fn == nil {
		return nil
	}
	conn, ok := tenantConnFrom(ctx)
	if !ok {
		return fn()
	}
	if bound := BoundTenantSchema(); bound != nil && bound.GetConnection() == conn {
		return fn()
	}
	s, err := TenantSchema(conn)
	if err != nil {
		return err
	}
	return WithBoundTenantSchema(s, fn)
}

// TenantSchema returns a fresh Schema on the tenant connection conn (safe to mutate).
func TenantSchema(conn string) (schema.Schema, error) {
	c, err := tenantdb.Default.Conn(conn)
	if err != nil {
		return nil, fmt.Errorf("tenant connection %s: %w", conn, err)
	}
	return c.NewSchema()
}

// schemaFor returns the Schema for ctx without binding anything: the tenant's when
// ctx carries a tenant connection (and the goroutine is not already bound to it),
// otherwise facades.Schema().
func schemaFor(ctx context.Context) (schema.Schema, error) {
	conn, ok := tenantConnFrom(ctx)
	if !ok {
		return Schema(), nil
	}
	if bound := BoundTenantSchema(); bound != nil && bound.GetConnection() == conn {
		return bound, nil
	}
	return TenantSchema(conn)
}

func tenantConnFrom(ctx context.Context) (string, bool) {
	if !tenancy.Enabled() || ctx == nil {
		return "", false
	}
	conn, ok := tenancyctx.ConnectionFrom(ctx)
	return conn, ok && conn != ""
}
