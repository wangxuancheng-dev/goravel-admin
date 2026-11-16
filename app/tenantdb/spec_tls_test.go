package tenantdb

import (
	"strings"
	"testing"
)

func TestMySQLDSNIncludesTLS(t *testing.T) {
	s := Spec{
		Driver: DriverMySQL, Host: "db.example", Port: 3306,
		Database: "tenant_acme", Username: "u", Password: "p", TLSMode: "true",
	}
	vals := s.connectionValues()
	dsn, _ := vals["dsn"].(string)
	if dsn == "" || !strings.Contains(dsn, "tls=true") {
		t.Fatalf("dsn=%q", dsn)
	}
	if !strings.Contains(dsn, "loc=") {
		t.Fatalf("loc missing: %q", dsn)
	}
	if vals["host"] != "db.example" {
		t.Fatalf("host still set for non-dsn fields: %v", vals["host"])
	}
}

func TestMySQLDSNOmittedWithoutTLS(t *testing.T) {
	s := Spec{Driver: DriverMySQL, Host: "127.0.0.1", Port: 3306, Database: "t", Username: "u", Password: "p"}
	vals := s.connectionValues()
	if _, ok := vals["dsn"]; ok {
		t.Fatal("dsn should be omitted when TLSMode empty")
	}
}

func TestFingerprintIncludesTLSMode(t *testing.T) {
	a := Spec{Driver: DriverMySQL, Host: "h", Port: 1, Database: "d", Username: "u", Password: "p"}
	b := a
	b.TLSMode = "true"
	if a.Fingerprint() == b.Fingerprint() {
		t.Fatal("TLSMode must change fingerprint")
	}
}
