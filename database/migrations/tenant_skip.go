package migrations

import (
	"strings"

	"github.com/goravel/framework/facades"
)

// SkipOnTenantConnection no-ops landlord-only migrations when Schema is bound to a tenant_* connection.
func SkipOnTenantConnection() bool {
	conn := strings.TrimSpace(facades.Schema().GetConnection())
	if conn == "" {
		conn = strings.TrimSpace(facades.Config().GetString("database.default", ""))
	}
	return strings.HasPrefix(conn, "tenant_")
}

// IsLandlordOnlyMigration reports migrations that must never create objects on tenant DBs.
func IsLandlordOnlyMigration(signature string) bool {
	switch strings.TrimSpace(signature) {
	case "20210101000002_create_jobs_table",
		"20260911000001_create_tenants_table",
		"20260911000002_create_platform_admins_table",
		"20260912000001_add_provision_status_to_tenants",
		"20260912000002_add_migrate_meta_to_tenants",
		"20260915140000_add_tenant_schema_migration_count",
		"20260915160000_add_tenant_maintenance_meta",
		"20260916120000_create_tenant_domains_table",
		"20260916140000_add_tenant_health_meta",
		"20260917220000_create_platform_login_logs_table",
		"20260917220001_create_platform_operation_logs_table",
		"20260920220000_platform_ops_extensions":
		return true
	default:
		return false
	}
}
