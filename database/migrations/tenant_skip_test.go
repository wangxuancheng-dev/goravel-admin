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
	if IsLandlordOnlyMigration("20250101000002_create_admins_table") {
		t.Fatal("admins should run on tenant DBs")
	}
}
