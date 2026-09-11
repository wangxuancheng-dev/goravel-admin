package middleware

import "testing"

func TestMatchPath(t *testing.T) {
	cases := []struct {
		pattern string
		path    string
		want    bool
	}{
		{"/api/admin/roles", "/api/admin/roles", true},
		{"/api/admin/roles", "/api/admin/admins", false},
		{"/api/admin/roles/*", "/api/admin/roles/1", true},
		{"/api/admin/roles/*", "/api/admin/roles", false},
		{"/api/admin/attachments/*/display-name", "/api/admin/attachments/9/display-name", true},
		{"/api/admin/attachments/*/display-name", "/api/admin/attachments/9/preview", false},
	}
	for _, tc := range cases {
		if got := matchPath(tc.pattern, tc.path); got != tc.want {
			t.Fatalf("matchPath(%q, %q)=%v want %v", tc.pattern, tc.path, got, tc.want)
		}
	}
}
