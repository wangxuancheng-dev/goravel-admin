package services

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/goravel/framework/contracts/database/orm"
	"github.com/goravel/framework/contracts/http"

	"goravel/app/models"
)

type TenantAdminFilters struct {
	Code            string
	Name            string
	Status          string
	ProvisionStatus string
	// SchemaStatus: aligned|behind|failed|running|unknown (computed vs current binary).
	SchemaStatus string
	// LastOp: migrate|seed|backup|restore|purge
	LastOp string
	// LastOpStatus: idle|queued|running|success|failed
	LastOpStatus string
	// Maintenance: "" = any; "1"/"true" = on; "0"/"false" = off.
	Maintenance string
	// Trashed: "" = exclude soft-deleted (default); "only" = recycle bin; "with" = include both.
	Trashed string
	// DomainHost: substring match on tenant_domains.host
	DomainHost string
	// DomainStatus: unbound|pending|active|verify_failed|disabled
	DomainStatus string
	// HealthStatus: ok|warn|fail|unknown
	HealthStatus string
	// PlanCode: platform_plans.code, or _none for no active subscription.
	PlanCode string
	// Expiring: 7|30 — active subscription ends_at within N days from now.
	Expiring string
}

func BuildTenantAdminFiltersFromHTTP(ctx http.Context) TenantAdminFilters {
	return TenantAdminFilters{
		Code:            strings.TrimSpace(ctx.Request().Query("code", "")),
		Name:            strings.TrimSpace(ctx.Request().Query("name", "")),
		Status:          strings.TrimSpace(ctx.Request().Query("status", "")),
		ProvisionStatus: strings.TrimSpace(ctx.Request().Query("provision_status", "")),
		SchemaStatus:    strings.TrimSpace(ctx.Request().Query("schema_status", "")),
		LastOp:          strings.TrimSpace(ctx.Request().Query("last_op", "")),
		LastOpStatus:    strings.TrimSpace(ctx.Request().Query("last_op_status", "")),
		Maintenance:     strings.TrimSpace(ctx.Request().Query("maintenance", "")),
		Trashed:         strings.TrimSpace(ctx.Request().Query("trashed", "")),
		DomainHost:      strings.TrimSpace(ctx.Request().Query("domain_host", "")),
		DomainStatus:    strings.TrimSpace(ctx.Request().Query("domain_status", "")),
		HealthStatus:    strings.TrimSpace(ctx.Request().Query("health_status", "")),
		PlanCode:        strings.TrimSpace(ctx.Request().Query("plan_code", "")),
		Expiring:        strings.TrimSpace(ctx.Request().Query("expiring", "")),
	}
}

// NormalizeTenantLastOpFilter returns a known op name or empty (ignore unknown).
func NormalizeTenantLastOpFilter(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case models.TenantOpMigrate, models.TenantOpSeed, models.TenantOpBackup, models.TenantOpRestore, models.TenantOpPurge, models.TenantOpRollback:
		return strings.ToLower(strings.TrimSpace(raw))
	default:
		return ""
	}
}

// NormalizeTenantLastOpStatusFilter returns a known status or empty (ignore unknown).
func NormalizeTenantLastOpStatusFilter(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case models.TenantOpStatusIdle, models.TenantOpStatusQueued, models.TenantOpStatusRunning,
		models.TenantOpStatusSuccess, models.TenantOpStatusFailed:
		return strings.ToLower(strings.TrimSpace(raw))
	default:
		return ""
	}
}

// pendingConnectionName is the unique placeholder connection_name of a tenant row
// until its id is known (tenant_<id> replaces it right after the insert). Derived
// from a hash of the code: "tenant_pending_"+code overflowed the 64-char column for
// codes over 49 chars (PostgreSQL rejected the insert, MySQL may truncate it).
func pendingConnectionName(code string) string {
	sum := sha256.Sum256([]byte(code))
	return "tenant_pending_" + hex.EncodeToString(sum[:12])
}

func applySchemaStatusFilter(query orm.Query, schemaStatus string) orm.Query {
	status := strings.ToLower(strings.TrimSpace(schemaStatus))
	if status == "" {
		return query
	}
	expected := ExpectedSchemaMigrationCount()
	switch status {
	case TenantSchemaRunning:
		return query.Where(
			"(provision_status = ? OR (last_op = ? AND last_op_status IN (?, ?)))",
			models.TenantProvisionMigrating,
			models.TenantOpMigrate,
			models.TenantOpStatusQueued,
			models.TenantOpStatusRunning,
		)
	case TenantSchemaFailed:
		return query.Where(
			"((last_migrate_error IS NOT NULL AND last_migrate_error != '') OR provision_status = ? OR (last_op = ? AND last_op_status = ?))",
			models.TenantProvisionFailed,
			models.TenantOpMigrate,
			models.TenantOpStatusFailed,
		)
	case TenantSchemaAligned:
		return query.Where("provision_status = ?", models.TenantProvisionReady).
			Where("schema_migration_count >= ?", expected).
			Where("(last_migrate_error IS NULL OR last_migrate_error = '')")
	case TenantSchemaUnknown:
		return query.Where("provision_status = ?", models.TenantProvisionReady).
			Where("schema_migration_count = ?", 0).
			Where("(last_migrate_error IS NULL OR last_migrate_error = '')")
	case TenantSchemaBehind:
		// Pending, or count below waterline, excluding failed/running/unknown-aligned cases.
		return query.Where(
			`(
				provision_status = ?
				OR (
					schema_migration_count < ?
					AND provision_status = ?
					AND schema_migration_count > 0
					AND (last_migrate_error IS NULL OR last_migrate_error = '')
				)
			)
			AND provision_status NOT IN (?, ?)
			AND NOT (last_op = ? AND last_op_status IN (?, ?))`,
			models.TenantProvisionPending,
			expected,
			models.TenantProvisionReady,
			models.TenantProvisionMigrating,
			models.TenantProvisionFailed,
			models.TenantOpMigrate,
			models.TenantOpStatusQueued,
			models.TenantOpStatusRunning,
		)
	default:
		return query
	}
}
