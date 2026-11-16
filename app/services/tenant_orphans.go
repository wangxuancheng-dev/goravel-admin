package services

import (
	"fmt"
	"sort"
	"strings"

	"github.com/goravel/framework/contracts/database/orm"
	"github.com/goravel/framework/facades"

	appfacades "goravel/app/facades"
	"goravel/app/models"
)

// Orphaned tenant storage: databases / schemas on the platform's database server that
// carry the tenant prefix but belong to no tenant row (trashed ones included). Before
// permanent deletes erased them, deleted tenants left their databases behind; a new
// tenant with the same code is refused (tenant_storage_exists) until it is gone.

// OrphanTenantStorage is one database (database isolation) or schema (PostgreSQL
// schema isolation in the platform database) that no tenant references.
type OrphanTenantStorage struct {
	Kind     string `json:"kind"`     // "database" | "schema"
	Name     string `json:"name"`     // database / schema name
	Code     string `json:"code"`     // tenant code the name was derived from
	Account  string `json:"account"`  // the matching dedicated DB account, when it exists
	Driver   string `json:"driver"`   // mysql | postgres
	Database string `json:"database"` // for schemas: the database holding it
}

// OrphanTenantAccounts lists dedicated-looking accounts (t_...) that no tenant uses
// and that belong to no orphaned database: reported, never dropped (an operator may
// have created an account with that name).
type OrphanTenantAccounts []string

// FindOrphanTenantStorage lists orphaned tenant databases / schemas on the platform's
// server, and t_ accounts no tenant uses. Conservative: any tenant row (trashed too,
// any host) naming a database, schema or account keeps it off the list.
func FindOrphanTenantStorage() ([]OrphanTenantStorage, OrphanTenantAccounts, error) {
	q := appfacades.PlatformQuery()
	if down, err := appfacades.QueryUnavailable(q); down {
		return nil, nil, err
	}
	var tenants []models.Tenant
	if err := q.Model(&models.Tenant{}).WithTrashed().Find(&tenants); err != nil {
		return nil, nil, err
	}
	usedDB, usedSchema, usedAccount := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, t := range tenants {
		usedDB[strings.ToLower(t.Database)] = true
		usedSchema[strings.ToLower(t.Schema)] = true
		usedAccount[t.Username] = true
		usedAccount[DedicatedTenantDBUsername(t.Code)] = true
	}

	driver := platformDriver()
	dbPrefix := strings.ToLower(facades.Config().GetString("tenancy.database_prefix", "tenant_"))
	schemaPrefix := strings.ToLower(facades.Config().GetString("tenancy.schema_prefix", "tenant_"))
	if dbPrefix == "" || schemaPrefix == "" {
		return nil, nil, fmt.Errorf("tenancy database/schema prefix is empty: refusing to guess which databases are tenants'")
	}
	platformDB := strings.ToLower(appfacades.ConnectionDatabaseName(appfacades.PlatformConnectionName()))

	accounts, err := listDedicatedLookingAccounts(driver)
	if err != nil {
		return nil, nil, err
	}
	accountExists := map[string]bool{}
	for _, a := range accounts {
		accountExists[a] = true
	}

	var out []OrphanTenantStorage
	add := func(kind, name, prefix, database string) {
		code := strings.TrimPrefix(name, prefix)
		if code == "" || validateSQLIdent(code) != nil {
			return
		}
		o := OrphanTenantStorage{Kind: kind, Name: name, Code: code, Driver: driver, Database: database}
		if acc := DedicatedTenantDBUsername(code); accountExists[acc] && !usedAccount[acc] {
			o.Account = acc
		}
		out = append(out, o)
	}

	dbNames, err := listNames(q, driver, "databases")
	if err != nil {
		return nil, nil, err
	}
	for _, n := range dbNames {
		ln := strings.ToLower(n)
		if strings.HasPrefix(ln, dbPrefix) && !usedDB[ln] && ln != platformDB {
			add("database", n, dbPrefix, n)
		}
	}
	if driver == models.TenantDriverPostgres {
		schemas, err := listNames(q, driver, "schemas")
		if err != nil {
			return nil, nil, err
		}
		for _, n := range schemas {
			ln := strings.ToLower(n)
			if strings.HasPrefix(ln, schemaPrefix) && !usedSchema[ln] {
				add("schema", n, schemaPrefix, platformDB)
			}
		}
	}

	paired := map[string]bool{}
	for _, o := range out {
		paired[o.Account] = true
	}
	var loose OrphanTenantAccounts
	for _, a := range accounts {
		if !usedAccount[a] && !paired[a] {
			loose = append(loose, a)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	sort.Strings(loose)
	return out, loose, nil
}

// DropOrphanTenantStorage drops one orphan found by FindOrphanTenantStorage (and its
// paired dedicated account) through DropStorage, with its safeguards: the tenancy
// prefix, never the platform database or a reserved schema. It re-checks that no
// tenant references the name first.
func DropOrphanTenantStorage(o OrphanTenantStorage) error {
	var n int64
	var err error
	if o.Kind == "schema" {
		n, err = appfacades.PlatformQuery().Model(&models.Tenant{}).WithTrashed().Where("schema", o.Name).Count()
	} else {
		n, err = appfacades.PlatformQuery().Model(&models.Tenant{}).WithTrashed().Where("database", o.Name).Count()
	}
	if err != nil {
		return err
	}
	if n > 0 {
		return fmt.Errorf("%s %s is now used by a tenant: not dropped", o.Kind, o.Name)
	}
	isolation := models.TenantIsolationDatabase
	tenant := &models.Tenant{Code: o.Code, Driver: o.Driver, Database: o.Database, Username: o.Account}
	if o.Kind == "schema" {
		isolation = models.TenantIsolationSchema
		tenant.Schema = o.Name
	}
	tenant.Isolation = isolation
	tenant.ConnectionName = "tenant_orphan_" + o.Code
	return NewTenantConnectionService().DropStorage(tenant)
}

func platformDriver() string {
	if isPostgresQuery(appfacades.PlatformQuery()) {
		return models.TenantDriverPostgres
	}
	return models.TenantDriverMySQL
}

func isPostgresQuery(q orm.Query) bool {
	return strings.Contains(strings.ToLower(q.Driver()), "postgres")
}

// listNames lists the server's databases, or (PostgreSQL) the schemas of the platform
// database.
func listNames(q orm.Query, driver, what string) ([]string, error) {
	var sql string
	switch {
	case driver == models.TenantDriverPostgres && what == "schemas":
		sql = "SELECT schema_name AS name FROM information_schema.schemata"
	case driver == models.TenantDriverPostgres:
		sql = "SELECT datname AS name FROM pg_database WHERE NOT datistemplate"
	default:
		sql = "SELECT schema_name AS name FROM information_schema.schemata"
	}
	return scanNames(q, sql)
}

// listDedicatedLookingAccounts lists accounts named like DedicatedTenantDBUsername
// (t_...). Needs read access to mysql.user / pg_roles; without it none are listed.
func listDedicatedLookingAccounts(driver string) ([]string, error) {
	sql := `SELECT DISTINCT user AS name FROM mysql.user WHERE user LIKE 't\_%'`
	if driver == models.TenantDriverPostgres {
		sql = `SELECT rolname AS name FROM pg_roles WHERE rolname LIKE 't\_%'`
	}
	names, err := scanNames(appfacades.PlatformQuery(), sql)
	if err != nil {
		facades.Log().Warningf("tenant orphans: cannot list DB accounts: %v", err)
		return nil, nil
	}
	return names, nil
}

func scanNames(q orm.Query, sql string) ([]string, error) {
	var rows []struct {
		Name string `gorm:"column:name"`
	}
	if err := q.Raw(sql).Scan(&rows); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.Name)
	}
	return out, nil
}
