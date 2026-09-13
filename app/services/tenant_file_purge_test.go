package services

import "testing"

func TestTenantObjectStoragePrefix(t *testing.T) {
	got, err := TenantObjectStoragePrefix("acme")
	if err != nil || got != "tenants/acme" {
		t.Fatalf("acme: got %q err=%v", got, err)
	}
	if _, err := TenantObjectStoragePrefix("../etc"); err == nil {
		t.Fatal("expected reject path traversal")
	}
	if _, err := TenantObjectStoragePrefix(""); err == nil {
		t.Fatal("expected reject empty")
	}
	if _, err := TenantObjectStoragePrefix("a/b"); err == nil {
		t.Fatal("expected reject slash")
	}
}
