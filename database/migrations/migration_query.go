package migrations

import (
	"github.com/goravel/framework/contracts/database/orm"
	"github.com/goravel/framework/facades"
)

// migrationQuery is the query data statements in a migration must use: the migrator
// runs each migration in a transaction and points the Schema's Orm at it.
//
// facades.Orm().Query() is another connection outside that transaction. On MySQL the
// difference hides (DDL commits implicitly), but on PostgreSQL DDL is transactional: a
// table created earlier in the migration is invisible to that connection, and touching
// a table the migration has altered waits on the migration's own lock.
func migrationQuery() orm.Query {
	return facades.Schema().Orm().Query()
}
