package config

import (
	"goravel/app/codegenerator"

	"github.com/goravel/framework/facades"
	"github.com/spf13/cast"
)

func init() {
	config := facades.Config()

	// Optional extras (comma-separated). Defaults live in app/codegenerator/reserved.go —
	// append built-in tables there; use env only for site-specific extras.
	extraTables := codegenerator.SplitCSV(cast.ToString(config.Env("CODE_GENERATOR_EXTRA_RESERVED_TABLES", "")))
	extraPrefixes := codegenerator.SplitCSV(cast.ToString(config.Env("CODE_GENERATOR_EXTRA_RESERVED_TABLE_PREFIXES", "")))

	config.Add("code_generator", map[string]any{
		// Frontend targets (also under module.code_generator_frontend for backward compat).
		"frontend": config.Env("CODE_GENERATOR_FRONTEND", "react,vue"),

		// Exact table names excluded from the generator table picker.
		"reserved_tables": codegenerator.MergeExtras(codegenerator.ReservedTables, extraTables),

		// Prefixes excluded from the generator table picker.
		"reserved_table_prefixes": codegenerator.MergeExtras(codegenerator.ReservedTablePrefixes, extraPrefixes),
	})
}
