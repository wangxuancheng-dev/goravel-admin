package services

import (
	"testing"

	apperrors "goravel/app/errors"
)

func TestParseOIDCStatePayload(t *testing.T) {
	got, err := parseOIDCStatePayload(`{"tenant_id":7,"tenant_code":"acme"}`)
	if err != nil || got.TenantID != 7 || got.TenantCode != "acme" {
		t.Fatalf("tenant payload: got %+v, %v", got, err)
	}
	if _, err := parseOIDCStatePayload(`{}`); err != nil {
		t.Fatalf("single-db payload {} rejected: %v", err)
	}
	// A bare "1" is what the claim marker holds on Redis; it must never pass as a state.
	for _, bad := range []string{"", "1", "true", "not json"} {
		if _, err := parseOIDCStatePayload(bad); err == nil {
			t.Errorf("parseOIDCStatePayload(%q) accepted a non-state value", bad)
		}
	}
}

func TestOIDCLoginEmail(t *testing.T) {
	cases := []struct {
		name            string
		email           string
		verified        any
		allowUnverified bool
		wantEmail       string
		wantCode        string
	}{
		{name: "verified bool", email: " Alice@Example.COM ", verified: true, wantEmail: "alice@example.com"},
		{name: "verified string", email: "a@example.com", verified: "TRUE", wantEmail: "a@example.com"},
		{name: "false rejected", email: "a@example.com", verified: false, wantCode: "oidc_email_unverified"},
		{name: "string false rejected", email: "a@example.com", verified: "false", wantCode: "oidc_email_unverified"},
		{name: "missing claim rejected", email: "a@example.com", verified: nil, wantCode: "oidc_email_unverified"},
		{name: "non-bool type rejected", email: "a@example.com", verified: float64(1), wantCode: "oidc_email_unverified"},
		{name: "missing claim allowed by switch", email: "a@example.com", verified: nil, allowUnverified: true, wantEmail: "a@example.com"},
		{name: "empty email", email: "  ", verified: true, wantCode: "oidc_email_required"},
		{name: "empty email even when unverified allowed", email: "", allowUnverified: true, wantCode: "oidc_email_required"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := oidcLoginEmail(tc.email, tc.verified, tc.allowUnverified)
			if tc.wantCode != "" {
				be, ok := apperrors.GetBusinessError(err)
				if !ok || be.Code != tc.wantCode {
					t.Fatalf("want error %q, got %v", tc.wantCode, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.wantEmail {
				t.Fatalf("email = %q, want %q", got, tc.wantEmail)
			}
		})
	}
}
