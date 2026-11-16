package services

import (
	"strings"
	"testing"
)

func TestDedicatedTenantDBUsername(t *testing.T) {
	if got := DedicatedTenantDBUsername("acme"); got != "t_acme" {
		t.Fatalf("short code: %q", got)
	}
	long := strings.Repeat("a", 40)
	got := DedicatedTenantDBUsername(long)
	if len(got) != mysqlMaxUserLen {
		t.Fatalf("long code: len %d (%q)", len(got), got)
	}
	if got == DedicatedTenantDBUsername(strings.Repeat("a", 39)+"b") {
		t.Fatal("codes sharing a long prefix must not share an account")
	}
	if err := validateSQLIdent(got); err != nil {
		t.Fatalf("not a valid identifier: %v", err)
	}
}

func TestGenerateDBPassword(t *testing.T) {
	a, err := generateDBPassword()
	if err != nil {
		t.Fatal(err)
	}
	b, _ := generateDBPassword()
	if len(a) != 32 || !isAlnum(a) || a == b {
		t.Fatalf("password %q / %q", a, b)
	}
}
