package services

import "testing"

func TestValidateChunkID(t *testing.T) {
	if err := validateChunkID("0123456789abcdef0123456789abcdef"); err != nil {
		t.Fatalf("valid chunk id rejected: %v", err)
	}
	for _, bad := range []string{
		"",
		"../etc/passwd",
		"0123456789abcdef0123456789abcde",
		"0123456789ABCDEF0123456789abcdef",
		"0123456789abcdef0123456789abcdefg",
		"chunks/../../x",
	} {
		if err := validateChunkID(bad); err == nil {
			t.Fatalf("expected reject for %q", bad)
		}
	}
}
