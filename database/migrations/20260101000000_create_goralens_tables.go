package migrations

import (
	"github.com/beratkara/goralens"
)

// M20260101000000CreateGoralensTables wraps the Goralens debug dashboard tables
// so they stay on the landlord DB. The signature is defined by the package.
//
// The package migration is held in a field, not embedded: embedding would promote
// its Connection() method, and the migrator calls SetConnection(m.Connection()) on
// any migration that has one. Even an empty name rebinds the Schema to
// database.default (the landlord), so a tenant migrate / rollback would run this
// migration on the landlord: SkipOnTenantConnection would no longer see the tenant,
// and a tenant rollback would drop the landlord's goralens tables.
type M20260101000000CreateGoralensTables struct {
	pkg goralens.Migration
}

func (m *M20260101000000CreateGoralensTables) Signature() string {
	return m.pkg.Signature()
}

func (m *M20260101000000CreateGoralensTables) Up() error {
	if SkipOnTenantConnection() {
		return nil
	}
	return m.pkg.Up()
}

func (m *M20260101000000CreateGoralensTables) Down() error {
	if SkipOnTenantConnection() {
		return nil
	}
	return m.pkg.Down()
}
