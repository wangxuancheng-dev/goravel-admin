package facades

import (
	contractsbinding "github.com/goravel/framework/contracts/binding"
	"github.com/goravel/framework/contracts/database/driver"
	"github.com/goravel/framework/contracts/database/orm"
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/contracts/foundation"
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
	app.Instance(contractsbinding.Schema, &schemaRouter{base: base})
	// Instance updates bindings only; clear the resolved singleton cache so MakeSchema returns the router.
	app.Fresh(contractsbinding.Schema)
}

func (r *schemaRouter) Connection(name string) schema.Schema {
	// Always create from the real singleton so callers get an independent Schema.
	// Serialize BuildQuery — framework MySQL config maps are not concurrent-safe.
	ormConnectionMu.Lock()
	defer ormConnectionMu.Unlock()
	return r.base.Connection(name)
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
	// SetConnection rebuilds via Orm.Connection — serialize like Connection().
	ormConnectionMu.Lock()
	defer ormConnectionMu.Unlock()
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
