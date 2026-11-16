package tenantdb

import (
	"errors"
	"fmt"
	"strings"
	"sync/atomic"

	gormio "gorm.io/gorm"
)

// ErrPlatformTable is returned by a tenant connection asked to read or write a
// platform-only table (see ForbidTables).
var ErrPlatformTable = errors.New("platform table on a tenant connection")

var forbiddenTables atomic.Pointer[map[string]bool]

// ForbidTables makes tenant connections refuse ORM statements (create / query /
// update / delete / row) whose table is one of tables: platform-only tables do not
// exist in tenant databases, so reaching one through OrmQuery(tenantCtx) is a routing
// bug — reported at once, naming PlatformQuery, instead of a "table not found" or,
// worse, a stale copy. Raw SQL (Raw / Exec) is not inspected.
//
// It applies to every tenant connection, including those already built.
func ForbidTables(tables []string) {
	m := make(map[string]bool, len(tables))
	for _, t := range tables {
		if t = normalizeTable(t); t != "" {
			m[t] = true
		}
	}
	forbiddenTables.Store(&m)
}

// checkTable returns ErrPlatformTable when table is forbidden.
func checkTable(table string) error {
	m := forbiddenTables.Load()
	if m == nil || len(*m) == 0 {
		return nil
	}
	if t := normalizeTable(table); (*m)[t] {
		return fmt.Errorf("%w: %q lives in the platform database; use appfacades.PlatformQuery() / PlatformOrmQuery(ctx), not OrmQuery(tenantCtx)", ErrPlatformTable, t)
	}
	return nil
}

// normalizeTable turns `"public"."tenants" AS t`, "`tenants` t" ... into "tenants".
func normalizeTable(table string) string {
	fields := strings.Fields(strings.TrimSpace(table))
	if len(fields) == 0 {
		return ""
	}
	t := fields[0]
	if i := strings.LastIndexByte(t, '.'); i >= 0 {
		t = t[i+1:]
	}
	return strings.ToLower(strings.Trim(t, "`\"[]"))
}

const guardCallback = "tenantdb:platform_table_guard"

// installTableGuard registers the ForbidTables check on a tenant gorm DB.
func installTableGuard(db *gormio.DB) error {
	guard := func(tx *gormio.DB) {
		if tx.Statement == nil {
			return
		}
		table := tx.Statement.Table
		if table == "" && tx.Statement.Schema != nil {
			table = tx.Statement.Schema.Table
		}
		if err := checkTable(table); err != nil {
			_ = tx.AddError(err)
		}
	}
	cb := db.Callback()
	for _, reg := range []func() error{
		func() error { return cb.Create().Before("gorm:create").Register(guardCallback, guard) },
		func() error { return cb.Query().Before("gorm:query").Register(guardCallback, guard) },
		func() error { return cb.Update().Before("gorm:update").Register(guardCallback, guard) },
		func() error { return cb.Delete().Before("gorm:delete").Register(guardCallback, guard) },
		func() error { return cb.Row().Before("gorm:row").Register(guardCallback, guard) },
	} {
		if err := reg(); err != nil {
			return err
		}
	}
	return nil
}
