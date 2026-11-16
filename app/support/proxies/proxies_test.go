package proxies

import "testing"

func TestParseAndContains(t *testing.T) {
	set := Parse("203.0.113.0/24, 198.51.100.9, ::1, bogus")
	if !set.Contains("203.0.113.200") {
		t.Fatal("cidr")
	}
	if !set.Contains("198.51.100.9") {
		t.Fatal("single ip")
	}
	if set.Contains("198.51.100.10") {
		t.Fatal("outside")
	}
	if !Parse("*").Contains("8.8.8.8") {
		t.Fatal("star")
	}
	if Parse("").Contains("127.0.0.1") {
		t.Fatal("empty")
	}
}

func TestParseIP(t *testing.T) {
	if got := ParseIP("203.0.113.7:443"); got != "203.0.113.7" {
		t.Fatalf("got %q", got)
	}
	if got := ParseIP("::1"); got != "127.0.0.1" {
		t.Fatalf("loopback: %q", got)
	}
}
