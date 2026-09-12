package middleware

import (
	"testing"

	"github.com/goravel/framework/contracts/http"
)

func TestIsForcePasswordChangeAllowed(t *testing.T) {
	cases := []struct {
		method string
		path   string
		want   bool
	}{
		{http.MethodGet, "/api/admin/info", true},
		{http.MethodGet, "/api/admin/heartbeat", true},
		{http.MethodPost, "/api/admin/logout", true},
		{http.MethodPut, "/api/admin/password", true},
		{http.MethodGet, "/api/admin/google-authenticator/status", true},
		{http.MethodPut, "/api/admin/admins/1/password", false},
		{http.MethodGet, "/api/admin/admins", false},
		{http.MethodPost, "/api/admin/admins", false},
	}
	for _, tc := range cases {
		if got := isForcePasswordChangeAllowed(tc.method, tc.path); got != tc.want {
			t.Fatalf("%s %s => %v, want %v", tc.method, tc.path, got, tc.want)
		}
	}
}
