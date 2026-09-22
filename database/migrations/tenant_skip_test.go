package migrations

import "testing"

func TestIsLandlordOnlyMigration(t *testing.T) {
	if !IsLandlordOnlyMigration("20260911000001_create_tenants_table") {
		t.Fatal("tenants table should be landlord-only")
	}
	if !IsLandlordOnlyMigration("20210101000002_create_jobs_table") {
		t.Fatal("jobs table should be landlord-only")
	}
	if !IsLandlordOnlyMigration("20260912000002_add_migrate_meta_to_tenants") {
		t.Fatal("migrate meta should be landlord-only")
	}
	if !IsLandlordOnlyMigration("20260916120000_create_tenant_domains_table") {
		t.Fatal("tenant_domains should be landlord-only")
	}
	if !IsLandlordOnlyMigration("20260921120000_create_flexible_schedules_table") {
		t.Fatal("flexible_schedules should be landlord-only")
	}
	if !IsLandlordOnlyMigration("20260922160000_flexible_schedules_payload_unique") {
		t.Fatal("flexible_schedules payload unique should be landlord-only")
	}
	if !IsLandlordOnlyMigration("20260922230000_add_flexible_schedule_next_run_at") {
		t.Fatal("flexible_schedules next_run_at should be landlord-only")
	}
	if IsLandlordOnlyMigration("20250101000002_create_admins_table") {
		t.Fatal("admins should run on tenant DBs")
	}
}
