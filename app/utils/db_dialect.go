package utils

import (
	"errors"

	"github.com/goravel/framework/contracts/database/orm"
	"gorm.io/gorm"
)

// Dialect-dependent SQL details are delegated to gorm's dialector behind a Goravel
// query (quoting, savepoints, error translation), so no driver name, quote character
// or error code is spelled out here.

// gormOf returns the gorm DB behind a Goravel query, or nil.
func gormOf(q orm.Query) *gorm.DB {
	if g, ok := q.(interface{ Instance() *gorm.DB }); ok {
		return g.Instance()
	}
	return nil
}

// QuoteIdent quotes a table / column name ("table.column" too) the way q's database
// expects (`name` on MySQL, "name" on PostgreSQL).
func QuoteIdent(q orm.Query, name string) string {
	if g := gormOf(q); g != nil {
		return g.Statement.Quote(name)
	}
	return name
}

// IsDuplicateKey reports a unique-key violation, as gorm's dialector translates it
// (gorm.ErrDuplicatedKey). The translators only recognise the bare driver error, so
// each error in the wrap chain is offered.
func IsDuplicateKey(q orm.Query, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	g := gormOf(q)
	if g == nil {
		return false
	}
	translator, ok := g.Dialector.(gorm.ErrorTranslator)
	if !ok {
		return false
	}
	for e := err; e != nil; e = errors.Unwrap(e) {
		if errors.Is(translator.Translate(e), gorm.ErrDuplicatedKey) {
			return true
		}
	}
	return false
}

// SavePoint sets a savepoint in q's transaction (gorm emits the dialect's statement).
func SavePoint(q orm.Query, name string) error {
	g := gormOf(q)
	if g == nil {
		return gorm.ErrUnsupportedDriver
	}
	// A session copy, so a failure is not recorded on the shared transaction handle.
	return g.Session(&gorm.Session{}).SavePoint(name).Error
}

// RollbackTo rolls q's transaction back to a savepoint set with SavePoint; the
// transaction stays usable (on PostgreSQL a failed statement otherwise aborts it).
func RollbackTo(q orm.Query, name string) error {
	g := gormOf(q)
	if g == nil {
		return gorm.ErrUnsupportedDriver
	}
	return g.Session(&gorm.Session{}).RollbackTo(name).Error
}
