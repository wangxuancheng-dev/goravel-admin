package tenantdb

import (
	"strings"

	"github.com/goravel/framework/contracts/database/driver"
	contractsschema "github.com/goravel/framework/contracts/database/schema"
)

// pgScopedSchema confines table lookups of a postgres tenant Schema to the
// connection's own schema (its search_path).
//
// The framework's postgres HasTable / GetTables list the tables of every schema in
// the database, so with schema isolation (several tenants in one database) a tenant
// would see another tenant's `migrations` or `admins` and skip creating its own. The
// listing also calls pg_total_relation_size on every table in the database, which
// deadlocks against other tenants' DDL when they migrate in parallel.
//
// Column / index lookups are already scoped to the connection's schema upstream.
//
// A migration may name another connection (the migrator then calls SetConnection on
// this Schema), in which case lookups go to the framework for that connection.
type pgScopedSchema struct {
	contractsschema.Schema
	conn   string
	schema string
	prefix string // the connection's table prefix, applied like the framework does
}

// ScopePostgresSchema wraps a framework Schema on postgres connection conn (search_path
// schema, table prefix) the same way, for connections tenantdb does not own: the
// platform's, whose database may also hold schema-isolated tenants.
func ScopePostgresSchema(s contractsschema.Schema, conn, schema, prefix string) contractsschema.Schema {
	if s == nil {
		return nil
	}
	if _, ok := s.(*pgScopedSchema); ok {
		return s
	}
	return newPgScopedSchema(s, conn, schema, prefix)
}

func newPgScopedSchema(s contractsschema.Schema, conn, schema, prefix string) contractsschema.Schema {
	if schema == "" {
		schema = "public"
	}
	return &pgScopedSchema{Schema: s, conn: conn, schema: schema, prefix: prefix}
}

// own reports whether the Schema still points at the tenant connection.
func (s *pgScopedSchema) own() bool {
	c := s.Schema.GetConnection()
	return c == "" || c == s.conn
}

// pgTablesSQL lists ordinary and partitioned tables of one schema. Sizes are only
// taken for that schema's tables, which this tenant owns.
const pgTablesSQL = "select c.relname as name, n.nspname as schema, pg_total_relation_size(c.oid) as size, " +
	"obj_description(c.oid, 'pg_class') as comment from pg_class c join pg_namespace n on n.oid = c.relnamespace " +
	"where c.relkind in ('r', 'p') and n.nspname = ? order by c.relname"

const pgHasTableSQL = "select count(*) from pg_class c join pg_namespace n on n.oid = c.relnamespace " +
	"where c.relkind in ('r', 'p') and n.nspname = ? and c.relname = ?"

// HasTable checks name in the connection's schema, or in the schema it names
// ("other.table"). It runs on the Schema's current query, so inside a migration it
// sees tables created earlier in the same transaction.
func (s *pgScopedSchema) HasTable(name string) bool {
	if !s.own() {
		return s.Schema.HasTable(name)
	}
	schema := s.schema
	if i := strings.LastIndex(name, "."); i >= 0 {
		schema, name = name[:i], name[i+1:]
	}
	var n int64
	if err := s.Orm().Query().Raw(pgHasTableSQL, schema, s.prefix+name).Scan(&n); err != nil {
		return false
	}
	return n > 0
}

func (s *pgScopedSchema) GetTables() ([]driver.Table, error) {
	if !s.own() {
		return s.Schema.GetTables()
	}
	var tables []driver.Table
	if err := s.Orm().Query().Raw(pgTablesSQL, s.schema).Scan(&tables); err != nil {
		return nil, err
	}
	return tables, nil
}

func (s *pgScopedSchema) GetTableListing() []string {
	if !s.own() {
		return s.Schema.GetTableListing()
	}
	tables, err := s.GetTables()
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(tables))
	for _, t := range tables {
		names = append(names, t.Name)
	}
	return names
}

var _ contractsschema.Schema = (*pgScopedSchema)(nil)
