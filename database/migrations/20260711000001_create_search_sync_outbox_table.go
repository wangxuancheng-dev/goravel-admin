package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
)

type M20260711000001CreateSearchSyncOutboxTable struct{}

func (r *M20260711000001CreateSearchSyncOutboxTable) Signature() string {
	return "20260711000001_create_search_sync_outbox_table"
}

func (r *M20260711000001CreateSearchSyncOutboxTable) Up() error {
	if facades.Schema().HasTable("search_sync_outbox") {
		return nil
	}
	return facades.Schema().Create("search_sync_outbox", func(table schema.Blueprint) {
		table.ID("id")
		table.String("entity_type", 32)
		table.UnsignedBigInteger("entity_id")
		table.String("entity_key", 64).Nullable()
		table.String("op", 16)
		table.Text("payload").Nullable()
		table.String("status", 16)
		table.UnsignedInteger("attempts").Default(0)
		table.Text("last_error").Nullable()
		table.Timestamp("processed_at").Nullable()
		table.Timestamps()

		table.Index("entity_type", "entity_id")
		table.Index("status", "attempts")
	})
}

func (r *M20260711000001CreateSearchSyncOutboxTable) Down() error {
	return facades.Schema().DropIfExists("search_sync_outbox")
}
