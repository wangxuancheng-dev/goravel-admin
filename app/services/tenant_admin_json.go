package services

import (
	"strings"

	"goravel/app/models"
	"goravel/app/tenantstorage"
)

func TenantToJSON(t *models.Tenant) map[string]any {
	return TenantToJSONWithDomain(t, TenantDomainListMeta{}.empty())
}

// TenantToJSONWithDomain includes vanity-domain list summary.
func TenantToJSONWithDomain(t *models.Tenant, domain TenantDomainListMeta) map[string]any {
	return TenantToJSONWithMeta(t, domain, TenantSubscriptionListMeta{})
}

// TenantToJSONWithMeta includes domain + active subscription summary.
func TenantToJSONWithMeta(t *models.Tenant, domain TenantDomainListMeta, sub TenantSubscriptionListMeta) map[string]any {
	if t == nil {
		return nil
	}
	var deletedAt any
	if t.DeletedAt.Valid {
		deletedAt = t.DeletedAt.Time
	}
	expected := ExpectedSchemaMigrationCount()
	if domain.ActiveHosts == nil {
		domain.ActiveHosts = []string{}
	}
	if domain.Status == "" {
		domain.Status = TenantDomainStatusUnbound
	}
	out := map[string]any{
		"id":                         t.ID,
		"code":                       t.Code,
		"name":                       t.Name,
		"status":                     t.Status,
		"provision_status":           t.ProvisionStatus,
		"driver":                     t.Driver,
		"isolation":                  t.Isolation,
		"host":                       t.Host,
		"port":                       t.Port,
		"database":                   t.Database,
		"schema":                     t.Schema,
		"username":                   t.Username,
		"has_password":               TenantHasPassword(t.Password),
		"connection_name":            t.ConnectionName,
		"last_migrate_error":         t.LastMigrateError,
		"migrated_at":                t.MigratedAt,
		"schema_migration_count":     t.SchemaMigrationCount,
		"expected_migration_count":   expected,
		"schema_status":              ResolveTenantSchemaStatus(t, expected),
		"maintenance":                t.Maintenance,
		"maintenance_message":        t.MaintenanceMessage,
		"last_op":                    t.LastOp,
		"last_op_status":             t.LastOpStatus,
		"last_op_message":            t.LastOpMessage,
		"last_op_at":                 t.LastOpAt,
		"last_backup_path":           t.LastBackupPath,
		"backup_dir":                 TenantBackupLocation(t.Code),
		"storage_limit_bytes":        t.StorageLimitBytes,
		"health_status":              strings.TrimSpace(t.HealthStatus),
		"health_checked_at":          t.HealthCheckedAt,
		"health_issues":              parseHealthIssues(t.HealthIssues),
		"last_ping_ok":               t.LastPingOK,
		"last_ping_ms":               t.LastPingMs,
		"domain_status":              domain.Status,
		"domain_primary_host":        domain.PrimaryHost,
		"domain_active_hosts":        domain.ActiveHosts,
		"domain_count":               domain.DomainCount,
		"plan_code":                  sub.PlanCode,
		"plan_name":                  sub.PlanName,
		"subscription_ends_at":       sub.EndsAt,
		"subscription_trial_ends_at": sub.TrialEndsAt,
		"has_active_subscription":    sub.HasActive,
		"deleted_at":                 deletedAt,
		"trashed":                    TenantIsTrashed(t),
		"created_at":                 t.CreatedAt,
		"updated_at":                 t.UpdatedAt,
	}
	for k, v := range tenantstorage.PublicJSON(t) {
		out[k] = v
	}
	return out
}

// TenantsToJSONList enriches rows with domain + subscription meta in batched queries.
func TenantsToJSONList(list []models.Tenant) []map[string]any {
	ids := make([]uint, 0, len(list))
	for i := range list {
		ids = append(ids, list[i].ID)
	}
	domainMeta := LoadTenantDomainListMeta(ids)
	subMeta := LoadTenantSubscriptionListMeta(ids)
	rows := make([]map[string]any, 0, len(list))
	for i := range list {
		rows = append(rows, TenantToJSONWithMeta(&list[i], domainMeta[list[i].ID], subMeta[list[i].ID]))
	}
	return rows
}
