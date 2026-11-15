package tenantcache

import (
	"reflect"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	"gorm.io/gorm"

	"goravel/app/models"
)

// Hook names, also used to make InstallHooks idempotent per gorm instance.
const (
	hookCreate = "tenantcache:after_create"
	hookUpdate = "tenantcache:after_update"
	hookDelete = "tenantcache:after_delete"
	hookRaw    = "tenantcache:after_raw"
)

var (
	// hooksInstalled gates the cache: without invalidation on writes, cached rows
	// could outlive a disabled or deleted tenant, so nothing is cached until
	// InstallHooks ran (and again after ResetHooks, until it runs again).
	hooksInstalled atomic.Bool

	// tableName is the tenants table as gorm names it (table prefix included).
	tableName atomic.Value // string
	// rawWrite matches raw SQL writing that table (whole word, so tenant_domains
	// does not match). A false positive only costs an extra invalidation.
	rawWrite atomic.Value // *regexp.Regexp
)

// txRecheckDelay re-invalidates after a write inside a transaction: the first
// invalidation runs before COMMIT, so a concurrent reader may cache the pre-commit
// row under the new version until then.
var txRecheckDelay = 2 * time.Second

// InstallHooks invalidates cached rows after every successful write to the tenants
// table on db (the platform connection's gorm), and enables the cache. Idempotent.
//
// The hooks live on that gorm instance: if it is replaced (Orm().Fresh()), call
// ResetHooks and InstallHooks again with the new one.
func InstallHooks(db *gorm.DB) {
	if db == nil {
		return
	}
	setTable(tenantsTableOf(db))

	cb := db.Callback()
	if cb.Create().Get(hookCreate) == nil {
		_ = cb.Create().After("gorm:create").Register(hookCreate, afterWrite)
	}
	if cb.Update().Get(hookUpdate) == nil {
		_ = cb.Update().After("gorm:update").Register(hookUpdate, afterWrite)
	}
	if cb.Delete().Get(hookDelete) == nil {
		_ = cb.Delete().After("gorm:delete").Register(hookDelete, afterWrite)
	}
	if cb.Raw().Get(hookRaw) == nil {
		_ = cb.Raw().After("gorm:raw").Register(hookRaw, afterRaw)
	}
	hooksInstalled.Store(true)
}

func init() { setTable("tenants") }

func setTable(name string) {
	tableName.Store(name)
	rawWrite.Store(regexp.MustCompile(`(?is)^\s*(update|delete|insert|replace)\b.*\b` + regexp.QuoteMeta(name) + `\b`))
}

// ResetHooks disables the cache until InstallHooks runs again (the gorm instance the
// hooks were on is being replaced). Rows cached so far are dropped as well.
func ResetHooks() {
	hooksInstalled.Store(false)
	InvalidateAll()
	l1.reset()
}

// tenantsTableOf asks gorm for the tenants table name, so a configured table prefix
// is honoured.
func tenantsTableOf(db *gorm.DB) string {
	st := &gorm.Statement{DB: db}
	if err := st.Parse(&models.Tenant{}); err == nil && st.Schema != nil && st.Schema.Table != "" {
		return st.Schema.Table
	}
	return "tenants"
}

func afterWrite(tx *gorm.DB) {
	if tx.Error != nil || tx.Statement == nil || !isTenantsTable(tableOf(tx.Statement)) {
		return
	}
	if id := tenantIDOf(tx.Statement); id != 0 {
		invalidate(tx, func() { Invalidate(id) })
		return
	}
	// Bulk write (Where(...).Update / Delete): the rows are not known here.
	invalidate(tx, InvalidateAll)
}

func afterRaw(tx *gorm.DB) {
	if tx.Error != nil || tx.Statement == nil {
		return
	}
	re, _ := rawWrite.Load().(*regexp.Regexp)
	if re != nil && re.MatchString(tx.Statement.SQL.String()) {
		invalidate(tx, InvalidateAll)
	}
}

func invalidate(tx *gorm.DB, fn func()) {
	fn()
	if _, inTx := tx.Statement.ConnPool.(gorm.TxCommitter); inTx {
		time.AfterFunc(txRecheckDelay, fn)
	}
}

func tableOf(st *gorm.Statement) string {
	if st.Table != "" {
		return st.Table
	}
	if st.Schema != nil {
		return st.Schema.Table
	}
	return ""
}

// isTenantsTable compares ignoring quotes and a schema / database qualifier
// (`tenants`, "public"."tenants", platform.tenants).
func isTenantsTable(name string) bool {
	want, _ := tableName.Load().(string)
	if want == "" {
		want = "tenants"
	}
	name = strings.TrimSpace(name)
	if i := strings.LastIndex(name, "."); i >= 0 {
		name = name[i+1:]
	}
	if fields := strings.Fields(name); len(fields) > 0 {
		name = fields[0] // "tenants AS t"
	}
	return strings.EqualFold(strings.Trim(name, "`\"[]"), want)
}

// tenantIDOf returns the id of the single tenant a statement writes through its model
// (Model(t).Update, Save(t), Delete(t), Create(&t)), or 0 for a bulk write.
func tenantIDOf(st *gorm.Statement) uint {
	for _, v := range []any{st.Model, st.Dest} {
		if t, ok := v.(*models.Tenant); ok && t != nil && t.ID != 0 {
			return t.ID
		}
	}
	if st.ReflectValue.IsValid() && st.ReflectValue.Kind() == reflect.Struct && st.ReflectValue.CanAddr() {
		if t, ok := st.ReflectValue.Addr().Interface().(*models.Tenant); ok && t.ID != 0 {
			return t.ID
		}
	}
	return 0
}
