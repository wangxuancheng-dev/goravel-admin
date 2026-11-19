package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
)

// M20261119000000CreateFeedbacksTable creates the feedbacks table (tenant migrate included).
type M20261119000000CreateFeedbacksTable struct{}

func (m *M20261119000000CreateFeedbacksTable) Signature() string {
	return "20261119000000_create_feedbacks_table"
}

func (m *M20261119000000CreateFeedbacksTable) Up() error {
	if facades.Schema().HasTable("feedbacks") {
		return nil
	}
	return facades.Schema().Create("feedbacks", func(table schema.Blueprint) {
		table.ID()
		table.String("contact", 128).Default("").Comment("contact or nickname")
		table.Text("content").Comment("feedback body")
		table.UnsignedTinyInteger("status").Default(0).Comment("0=open 1=done")
		table.UnsignedBigInteger("created_by").Default(0).Comment("host actor id when known")
		table.Timestamps()
		table.SoftDeletes()
		table.Index("status")
		table.Index("created_by")
	})
}

func (m *M20261119000000CreateFeedbacksTable) Down() error {
	return facades.Schema().DropIfExists("feedbacks")
}
