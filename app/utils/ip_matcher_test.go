package utils

import "testing"

func TestIsIPInBlacklistLoopbackEquivalent(t *testing.T) {
	cases := []struct {
		ip, pattern string
	}{
		{"::1", "127.0.0.1"},
		{"127.0.0.1", "::1"},
		{"::1", "::1"},
		{"127.0.0.1", "127.0.0.1"},
		{"::1", "127.0.0.0/8"},
	}
	for _, tc := range cases {
		if !IsIPInBlacklist(tc.ip, tc.pattern) {
			t.Fatalf("expected match ip=%q pattern=%q", tc.ip, tc.pattern)
		}
	}
}
