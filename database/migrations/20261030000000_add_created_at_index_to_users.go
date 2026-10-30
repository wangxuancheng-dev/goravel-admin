package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
)

// M20261030000000AddCreatedAtIndexToUsers backs the user list's default ORDER BY created_at DESC;
// without it every page sorts the whole table.
type M20261030000000AddCreatedAtIndexToUsers struct{}

// Signature The unique signature for the migration.
func (r *M20261030000000AddCreatedAtIndexToUsers) Signature() string {
	return "20261030000000_add_created_at_index_to_users"
}

// Up Run the migrations.
func (r *M20261030000000AddCreatedAtIndexToUsers) Up() error {
	if !facades.Schema().HasTable("users") || facades.Schema().HasIndex("users", "users_created_at_index") {
		return nil
	}
	return facades.Schema().Table("users", func(table schema.Blueprint) {
		table.Index("created_at")
	})
}

// Down Reverse the migrations.
func (r *M20261030000000AddCreatedAtIndexToUsers) Down() error {
	if !facades.Schema().HasTable("users") || !facades.Schema().HasIndex("users", "users_created_at_index") {
		return nil
	}
	return facades.Schema().Table("users", func(table schema.Blueprint) {
		table.DropIndex("created_at")
	})
}
