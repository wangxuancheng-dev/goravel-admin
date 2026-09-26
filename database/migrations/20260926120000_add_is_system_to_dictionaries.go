package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
)

type M20260926120000AddIsSystemToDictionaries struct{}

func (r *M20260926120000AddIsSystemToDictionaries) Signature() string {
	return "20260926120000_add_is_system_to_dictionaries"
}

func (r *M20260926120000AddIsSystemToDictionaries) Up() error {
	if !facades.Schema().HasTable("dictionaries") {
		return nil
	}

	columns, err := facades.Schema().GetColumns("dictionaries")
	if err != nil {
		return err
	}
	hasColumn := false
	for _, column := range columns {
		if column.Name == "is_system" {
			hasColumn = true
			break
		}
	}

	if !hasColumn {
		if err := facades.Schema().Table("dictionaries", func(table schema.Blueprint) {
			table.UnsignedTinyInteger("is_system").Default(0).Comment("system dictionary 1:yes 0:no")
		}); err != nil {
			return err
		}
	}

	return r.markBuiltinSystem()
}

func (r *M20260926120000AddIsSystemToDictionaries) markBuiltinSystem() error {
	_, err := facades.Orm().Query().Table("dictionaries").
		WhereIn("type", []any{"status", "menu_type"}).
		Update(map[string]any{"is_system": 1})
	return err
}

func (r *M20260926120000AddIsSystemToDictionaries) Down() error {
	if !facades.Schema().HasTable("dictionaries") {
		return nil
	}
	return facades.Schema().Table("dictionaries", func(table schema.Blueprint) {
		table.DropColumn("is_system")
	})
}