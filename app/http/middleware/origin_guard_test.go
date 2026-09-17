package middleware

import (
	"testing"

	"github.com/goravel/framework/facades"
)

func TestIsAdminOriginHostAllowedConfigAndBase(t *testing.T) {
	facades.Config().Add("tenancy.driver", "off")
	facades.Config().Add("tenancy.base_domain", "example.com")

	cases := []struct {
		host   string
		config []string
		want   bool
	}{
		{"admin.example.com", []string{"admin.example.com"}, true},
		{"api.example.com", []string{"api.example.com", "*.example.com"}, true},
		{"acme.example.com", nil, true}, // TENANCY_BASE_DOMAIN
		{"example.com", nil, true},
		{"evil.com", nil, false},
		{"evil.com", []string{"admin.example.com"}, false},
		{"shop.customer.com", []string{"*.customer.com"}, true},
	}

	for _, tc := range cases {
		if got := isAdminOriginHostAllowed(tc.host, tc.config); got != tc.want {
			t.Fatalf("isAdminOriginHostAllowed(%q)=%v want %v", tc.host, got, tc.want)
		}
	}
}
