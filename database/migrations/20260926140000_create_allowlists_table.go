package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
)

type M20260926140000CreateAllowlistsTable struct {
}

func (r *M20260926140000CreateAllowlistsTable) Signature() string {
	return "20260926140000_create_allowlists_table"
}

func (r *M20260926140000CreateAllowlistsTable) Up() error {
	if !facades.Schema().HasTable("allowlists") {
		return facades.Schema().Create("allowlists", func(table schema.Blueprint) {
			table.BigIncrements("id")
			table.String("ip").Default("").Comment("IP or CIDR / range, comma-separated")
			table.String("remark").Nullable().Comment("remark")
			table.TinyInteger("status").Default(1).Comment("1 enabled 0 disabled")
			table.Timestamps()
			table.Index("ip")
			table.Index("status")
			table.Comment("IP allowlist")
		})
	}
	return nil
}

func (r *M20260926140000CreateAllowlistsTable) Down() error {
	return facades.Schema().DropIfExists("allowlists")
}
