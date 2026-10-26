package migrations

import (
	"github.com/beratkara/goralens"
)

// M20260101000000CreateGoralensTables wraps the Goralens debug dashboard tables
// so they stay on the landlord DB. The signature is defined by the package.
type M20260101000000CreateGoralensTables struct {
	goralens.Migration
}

// Connection stays empty like every other migration: the package would return
// goralens.storage.connection, making the migrator rebind a tenant Schema to
// the landlord connection and defeating SkipOnTenantConnection.
func (m *M20260101000000CreateGoralensTables) Connection() string {
	return ""
}

func (m *M20260101000000CreateGoralensTables) Up() error {
	if SkipOnTenantConnection() {
		return nil
	}
	return m.Migration.Up()
}

func (m *M20260101000000CreateGoralensTables) Down() error {
	if SkipOnTenantConnection() {
		return nil
	}
	return m.Migration.Down()
}
