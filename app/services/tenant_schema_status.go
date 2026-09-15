package services

import (
	"strings"

	"goravel/app/models"
	dbmigrations "goravel/database/migrations"
)

const (
	TenantSchemaAligned = "aligned"
	TenantSchemaBehind  = "behind"
	TenantSchemaFailed  = "failed"
	TenantSchemaRunning = "running"
	TenantSchemaUnknown = "unknown"
)

// ExpectedSchemaMigrationCount is the waterline of the running binary.
func ExpectedSchemaMigrationCount() int64 {
	return dbmigrations.RegisteredCount()
}

// ResolveTenantSchemaStatus classifies a tenant relative to the current binary.
func ResolveTenantSchemaStatus(t *models.Tenant, expected int64) string {
	if t == nil {
		return TenantSchemaUnknown
	}
	ps := strings.TrimSpace(t.ProvisionStatus)
	op := strings.TrimSpace(t.LastOp)
	os := strings.TrimSpace(t.LastOpStatus)

	if ps == models.TenantProvisionMigrating ||
		(op == models.TenantOpMigrate && (os == models.TenantOpStatusQueued || os == models.TenantOpStatusRunning)) {
		return TenantSchemaRunning
	}
	if strings.TrimSpace(t.LastMigrateError) != "" ||
		ps == models.TenantProvisionFailed ||
		(op == models.TenantOpMigrate && os == models.TenantOpStatusFailed) {
		return TenantSchemaFailed
	}
	if expected > 0 && t.SchemaMigrationCount >= expected && ps == models.TenantProvisionReady {
		return TenantSchemaAligned
	}
	if t.SchemaMigrationCount == 0 && ps == models.TenantProvisionReady {
		// Legacy row: migrated before schema_migration_count existed.
		return TenantSchemaUnknown
	}
	if ps == models.TenantProvisionPending || t.SchemaMigrationCount < expected {
		return TenantSchemaBehind
	}
	return TenantSchemaBehind
}

// SchemaStatusCounts aggregates schema_status over tenants (non-trashed).
type SchemaStatusCounts struct {
	Expected int64            `json:"expected_migration_count"`
	Aligned  int64            `json:"aligned"`
	Behind   int64            `json:"behind"`
	Failed   int64            `json:"failed"`
	Running  int64            `json:"running"`
	Unknown  int64            `json:"unknown"`
	ByStatus map[string]int64 `json:"by_schema_status"`
}

func CountSchemaStatuses(tenants []models.Tenant) SchemaStatusCounts {
	expected := ExpectedSchemaMigrationCount()
	out := SchemaStatusCounts{
		Expected: expected,
		ByStatus: map[string]int64{},
	}
	for i := range tenants {
		st := ResolveTenantSchemaStatus(&tenants[i], expected)
		out.ByStatus[st]++
		switch st {
		case TenantSchemaAligned:
			out.Aligned++
		case TenantSchemaBehind:
			out.Behind++
		case TenantSchemaFailed:
			out.Failed++
		case TenantSchemaRunning:
			out.Running++
		default:
			out.Unknown++
		}
	}
	return out
}
