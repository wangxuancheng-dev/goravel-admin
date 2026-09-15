package services

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"goravel/app/models"
)

func TestResolveTenantSchemaStatus(t *testing.T) {
	expected := int64(80)

	t.Run("running", func(t *testing.T) {
		assert.Equal(t, TenantSchemaRunning, ResolveTenantSchemaStatus(&models.Tenant{
			ProvisionStatus: models.TenantProvisionMigrating,
		}, expected))
		assert.Equal(t, TenantSchemaRunning, ResolveTenantSchemaStatus(&models.Tenant{
			LastOp:       models.TenantOpMigrate,
			LastOpStatus: models.TenantOpStatusQueued,
		}, expected))
	})

	t.Run("failed", func(t *testing.T) {
		assert.Equal(t, TenantSchemaFailed, ResolveTenantSchemaStatus(&models.Tenant{
			LastMigrateError: "boom",
			ProvisionStatus:  models.TenantProvisionReady,
			SchemaMigrationCount: expected,
		}, expected))
		assert.Equal(t, TenantSchemaFailed, ResolveTenantSchemaStatus(&models.Tenant{
			ProvisionStatus: models.TenantProvisionFailed,
		}, expected))
	})

	t.Run("aligned", func(t *testing.T) {
		assert.Equal(t, TenantSchemaAligned, ResolveTenantSchemaStatus(&models.Tenant{
			ProvisionStatus:      models.TenantProvisionReady,
			SchemaMigrationCount: expected,
		}, expected))
	})

	t.Run("behind", func(t *testing.T) {
		assert.Equal(t, TenantSchemaBehind, ResolveTenantSchemaStatus(&models.Tenant{
			ProvisionStatus:      models.TenantProvisionPending,
			SchemaMigrationCount: 0,
		}, expected))
		assert.Equal(t, TenantSchemaBehind, ResolveTenantSchemaStatus(&models.Tenant{
			ProvisionStatus:      models.TenantProvisionReady,
			SchemaMigrationCount: expected - 1,
		}, expected))
	})

	t.Run("unknown legacy", func(t *testing.T) {
		assert.Equal(t, TenantSchemaUnknown, ResolveTenantSchemaStatus(&models.Tenant{
			ProvisionStatus:      models.TenantProvisionReady,
			SchemaMigrationCount: 0,
		}, expected))
	})
}
