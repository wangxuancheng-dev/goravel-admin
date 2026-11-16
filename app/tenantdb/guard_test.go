package tenantdb

import (
	"errors"
	"testing"
)

func TestNormalizeTable(t *testing.T) {
	for in, want := range map[string]string{
		"tenants":                 "tenants",
		"  Tenants  ":             "tenants",
		"`tenants` t":             "tenants",
		`"public"."tenants" AS t`: "tenants",
		"goravel.platform_admins": "platform_admins",
		"":                        "",
	} {
		if got := normalizeTable(in); got != want {
			t.Errorf("normalizeTable(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCheckTable(t *testing.T) {
	t.Cleanup(func() { ForbidTables(nil) })

	if err := checkTable("tenants"); err != nil {
		t.Fatalf("nothing forbidden yet: %v", err)
	}
	ForbidTables([]string{"tenants", "platform_admins"})
	if err := checkTable("`tenants` AS t"); !errors.Is(err, ErrPlatformTable) {
		t.Fatalf("tenants: err = %v", err)
	}
	if err := checkTable("admins"); err != nil {
		t.Fatalf("admins: %v", err)
	}
	if err := checkTable(""); err != nil {
		t.Fatalf("no table: %v", err)
	}
}
