package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
)

type M20260911000003AlterTenantsPasswordLength struct{}

func (r *M20260911000003AlterTenantsPasswordLength) Signature() string {
	return "20260911000003_alter_tenants_password_length"
}

func (r *M20260911000003AlterTenantsPasswordLength) Up() error {
	return facades.Schema().Table("tenants", func(table schema.Blueprint) {
		// Crypt payloads exceed 255; store sealed secrets safely.
		table.String("password", 1024).Nullable().Change()
	})
}

func (r *M20260911000003AlterTenantsPasswordLength) Down() error {
	return facades.Schema().Table("tenants", func(table schema.Blueprint) {
		table.String("password", 255).Nullable().Change()
	})
}
