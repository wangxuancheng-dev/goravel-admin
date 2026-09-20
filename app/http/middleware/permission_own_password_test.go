package middleware

import "testing"

func TestIsOwnPasswordUpdate(t *testing.T) {
	cases := []struct {
		method, path string
		want         bool
	}{
		{"PUT", "/api/admin/password", true},
		{"PUT", "/api/admin/password/", true},
		{"PUT", "/api/admin/admins/1/password", false},
		{"GET", "/api/admin/password", false},
		{"POST", "/api/admin/password", false},
	}
	for _, tc := range cases {
		if got := isOwnPasswordUpdate(tc.method, tc.path); got != tc.want {
			t.Fatalf("isOwnPasswordUpdate(%q,%q)=%v want %v", tc.method, tc.path, got, tc.want)
		}
	}
}
