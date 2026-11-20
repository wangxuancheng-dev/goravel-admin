package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
)

// M20261120000001AddImpersonatorToOperationLogs tags rows written during a platform
// "sign in as tenant admin" session with the platform account that acted.
type M20261120000001AddImpersonatorToOperationLogs struct{}

func (r *M20261120000001AddImpersonatorToOperationLogs) Signature() string {
	return "20261120000001_add_impersonator_to_operation_logs"
}

func (r *M20261120000001AddImpersonatorToOperationLogs) Up() error {
	if !facades.Schema().HasTable("operation_logs") || facades.Schema().HasColumn("operation_logs", "impersonator") {
		return nil
	}
	return facades.Schema().Table("operation_logs", func(table schema.Blueprint) {
		table.String("impersonator", 120).Default("").Comment("平台代管账号（空=本人操作）")
	})
}

func (r *M20261120000001AddImpersonatorToOperationLogs) Down() error {
	if !facades.Schema().HasTable("operation_logs") || !facades.Schema().HasColumn("operation_logs", "impersonator") {
		return nil
	}
	return facades.Schema().Table("operation_logs", func(table schema.Blueprint) {
		table.DropColumn("impersonator")
	})
}
