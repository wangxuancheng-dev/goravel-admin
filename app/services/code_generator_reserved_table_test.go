package services

import (
	"testing"

	"goravel/app/codegenerator"
)

func TestIsCodeGeneratorReservedTable(t *testing.T) {
	reserved := []string{
		"admins", "users", "roles", "menus", "orders", "order_details",
		"payments", "tenants", "platform_admins", "platform_login_logs",
		"tenant_domains", "orders_202501", "user_balance_logs_0", "migrations",
	}
	for _, name := range reserved {
		if !codegenerator.IsReservedTable(name) {
			t.Fatalf("expected reserved: %s", name)
		}
		if !IsCodeGeneratorReservedTable(name) {
			t.Fatalf("service wrapper expected reserved: %s", name)
		}
	}

	allowed := []string{"articles", "quotes", "quote_items", "products", "guestbook_messages"}
	for _, name := range allowed {
		if codegenerator.IsReservedTable(name) {
			t.Fatalf("expected allowed: %s", name)
		}
		if IsCodeGeneratorReservedTable(name) {
			t.Fatalf("service wrapper expected allowed: %s", name)
		}
	}
}

func TestMergeExtrasReservedTables(t *testing.T) {
	merged := codegenerator.MergeExtras(codegenerator.ReservedTables, []string{"custom_sys", "admins", ""})
	found := false
	for _, tname := range merged {
		if tname == "custom_sys" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected custom_sys in merged reserved tables")
	}
	if !codegenerator.IsReservedTableWith("custom_sys", merged, nil) {
		t.Fatal("merged list should reserve custom_sys")
	}
}
