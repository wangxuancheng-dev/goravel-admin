package facades

import (
	"errors"

	contractsbinding "github.com/goravel/framework/contracts/binding"
	"github.com/goravel/framework/contracts/database/driver"
	"github.com/goravel/framework/contracts/database/orm"
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/contracts/foundation"
	goravelpostgres "github.com/goravel/postgres"

	"goravel/app/tenantdb"
)

// schemaRouter delegates to a goroutine-bound tenant Schema when set
// (parallel MigrateTenant), otherwise to the framework singleton.
type schemaRouter struct {
	base schema.Schema
}

func (r *schemaRouter) active() schema.Schema {
	if s := BoundTenantSchema(); s != nil {
		return s
	}
	return r.base
}

// InstallTenantAwareSchema wraps the container Schema so facades.Schema()
// honors BindTenantSchema during parallel tenant migrate.
func InstallTenantAwareSchema(app foundation.Application) {
	if app == nil {
		return
	}
	base := app.MakeSchema()
	if base == nil {
		return
	}
	if _, ok := base.(*schemaRouter); ok {
		return
	}
	app.Instance(contractsbinding.Schema, &schemaRouter{base: scopeIfPostgres(base)})
	// Instance updates bindings only; clear the resolved singleton cache so MakeSchema returns the router.
	app.Fresh(contractsbinding.Schema)
}

// Connection returns an independent Schema for name: a fresh one over the app-owned
// pool for tenant connections, the framework's for others.
func (r *schemaRouter) Connection(name string) schema.Schema {
	c, err := tenantdb.Default.Conn(name)
	if err == nil {
		s, schemaErr := c.NewSchema()
		if schemaErr != nil {
			logTenantConnError(name, schemaErr)
			return nil
		}
		return s
	}
	if !errors.Is(err, tenantdb.ErrUnknownConnection) {
		logTenantConnError(name, err)
		return nil
	}
	return scopeIfPostgres(r.base.Connection(name))
}

// scopeIfPostgres confines table lookups of a non-tenant postgres Schema (the
// platform's) to its own schema: with schema isolation, tenants' schemas live in the
// platform database by default, and the framework's HasTable / GetTables would report
// their tables as the platform's.
func scopeIfPostgres(s schema.Schema) schema.Schema {
	if s == nil || s.Orm() == nil {
		return s
	}
	// Query() is nil when the framework could not build the connection: leave the
	// Schema as it is, its own calls report that error (do not panic here).
	if q := s.Orm().Query(); q == nil || q.Driver() != goravelpostgres.Name {
		return s
	}
	// The schema and prefix the framework itself applies for this connection.
	cfg := s.Orm().Config()
	return tenantdb.ScopePostgresSchema(s, s.GetConnection(), cfg.Schema, cfg.Prefix)
}

func (r *schemaRouter) Create(table string, callback func(table schema.Blueprint)) error {
	return r.active().Create(table, callback)
}

func (r *schemaRouter) Drop(table string) error {
	return r.active().Drop(table)
}

func (r *schemaRouter) DropAllTables() error {
	return r.active().DropAllTables()
}

func (r *schemaRouter) DropAllTypes() error {
	return r.active().DropAllTypes()
}

func (r *schemaRouter) DropAllViews() error {
	return r.active().DropAllViews()
}

func (r *schemaRouter) DropColumns(table string, columns []string) error {
	return r.active().DropColumns(table, columns)
}

func (r *schemaRouter) DropIfExists(table string) error {
	return r.active().DropIfExists(table)
}

func (r *schemaRouter) Extend(extend schema.Extension) schema.Schema {
	return r.active().Extend(extend)
}

func (r *schemaRouter) GetColumnListing(table string) []string {
	return r.active().GetColumnListing(table)
}

func (r *schemaRouter) GetColumns(table string) ([]driver.Column, error) {
	return r.active().GetColumns(table)
}

func (r *schemaRouter) GetConnection() string {
	return r.active().GetConnection()
}

func (r *schemaRouter) GetForeignKeys(table string) ([]driver.ForeignKey, error) {
	return r.active().GetForeignKeys(table)
}

func (r *schemaRouter) GetIndexListing(table string) []string {
	return r.active().GetIndexListing(table)
}

func (r *schemaRouter) GetIndexes(table string) ([]driver.Index, error) {
	return r.active().GetIndexes(table)
}

func (r *schemaRouter) GetModel(name string) any {
	return r.active().GetModel(name)
}

func (r *schemaRouter) GetTableListing() []string {
	return r.active().GetTableListing()
}

func (r *schemaRouter) GetTables() ([]driver.Table, error) {
	return r.active().GetTables()
}

func (r *schemaRouter) GetTypes() ([]driver.Type, error) {
	return r.active().GetTypes()
}

func (r *schemaRouter) GetViews() ([]driver.View, error) {
	return r.active().GetViews()
}

func (r *schemaRouter) GoTypes() []schema.GoType {
	return r.active().GoTypes()
}

func (r *schemaRouter) HasColumn(table, column string) bool {
	return r.active().HasColumn(table, column)
}

func (r *schemaRouter) HasColumns(table string, columns []string) bool {
	return r.active().HasColumns(table, columns)
}

func (r *schemaRouter) HasIndex(table, index string) bool {
	return r.active().HasIndex(table, index)
}

func (r *schemaRouter) HasTable(name string) bool {
	return r.active().HasTable(name)
}

func (r *schemaRouter) HasType(name string) bool {
	return r.active().HasType(name)
}

func (r *schemaRouter) HasView(name string) bool {
	return r.active().HasView(name)
}

func (r *schemaRouter) Migrations() []schema.Migration {
	return r.base.Migrations()
}

func (r *schemaRouter) Orm() orm.Orm {
	return r.active().Orm()
}

func (r *schemaRouter) Prune() error {
	return r.active().Prune()
}

func (r *schemaRouter) Register(migrations []schema.Migration) {
	r.base.Register(migrations)
}

func (r *schemaRouter) Rename(from, to string) error {
	return r.active().Rename(from, to)
}

func (r *schemaRouter) SetConnection(name string) {
	if s := BoundTenantSchema(); s != nil {
		s.SetConnection(name)
		return
	}
	r.base.SetConnection(name)
}

func (r *schemaRouter) Sql(sql string) error {
	return r.active().Sql(sql)
}

func (r *schemaRouter) Table(table string, callback func(table schema.Blueprint)) error {
	return r.active().Table(table, callback)
}

var _ schema.Schema = (*schemaRouter)(nil)
