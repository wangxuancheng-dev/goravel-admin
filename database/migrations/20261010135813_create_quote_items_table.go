package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
)

type M20261010135813CreateQuoteItemTable struct {
}

func (m *M20261010135813CreateQuoteItemTable) Signature() string {
	return "20261010135813_create_quote_items_table"
}

func (m *M20261010135813CreateQuoteItemTable) Up() error {
	return facades.Schema().Create("quote_items", func(table schema.Blueprint) {
		table.ID()

		table.BigInteger("quote_id").Comment("quotes.id")
		table.String("product_name").Comment("product name")
		table.Integer("quantity").Comment("qty")
		table.Decimal("unit_price").Total(10).Places(2).Comment("unit price")
		table.Timestamps()
		table.SoftDeletes()
	})
}
func (m *M20261010135813CreateQuoteItemTable) Down() error {
	return facades.Schema().DropIfExists("quote_items")
}
