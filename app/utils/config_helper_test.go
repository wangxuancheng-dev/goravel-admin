package utils

import "testing"

func TestParseConfigBool(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"1", true},
		{"true", true},
		{"TRUE", true},
		{"on", true},
		{"yes", true},
		{"0", false},
		{"false", false},
		{"", false},
		{" 0 ", false},
		{"off", false},
	}
	for _, tc := range cases {
		if got := ParseConfigBool(tc.in); got != tc.want {
			t.Fatalf("ParseConfigBool(%q)=%v want %v", tc.in, got, tc.want)
		}
	}
}
