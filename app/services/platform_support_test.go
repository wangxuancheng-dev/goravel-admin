package services

import "testing"

func TestIPInAllowlist(t *testing.T) {
	if !IPInAllowlist("1.2.3.4", "") {
		t.Fatal("empty allowlist should allow")
	}
	if !IPInAllowlist("1.2.3.4", "1.2.3.4") {
		t.Fatal("exact match")
	}
	if IPInAllowlist("1.2.3.5", "1.2.3.4") {
		t.Fatal("exact mismatch")
	}
	if !IPInAllowlist("10.0.0.8", "10.0.0.0/24") {
		t.Fatal("cidr match")
	}
	if IPInAllowlist("10.0.1.8", "10.0.0.0/24") {
		t.Fatal("cidr mismatch")
	}
	if !IPInAllowlist("8.8.8.8", "1.1.1.1, 8.8.8.8 ,9.9.9.9") {
		t.Fatal("list match")
	}
}
