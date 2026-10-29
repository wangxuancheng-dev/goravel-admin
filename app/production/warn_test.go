package production

import (
	"testing"

	"github.com/goravel/framework/facades"
)

func TestWarnInsecureDefaultsNoPanicOutsideProduction(t *testing.T) {
	prev := facades.Config().GetString("app.env")
	facades.Config().Add("app.env", "local")
	t.Cleanup(func() { facades.Config().Add("app.env", prev) })
	WarnInsecureDefaults() // must not panic
}

func TestWarnInsecureDefaultsRunsInProduction(t *testing.T) {
	prevEnv := facades.Config().GetString("app.env")
	prevDebug := facades.Config().GetBool("app.debug")
	facades.Config().Add("app.env", "production")
	facades.Config().Add("app.debug", true)
	t.Cleanup(func() {
		facades.Config().Add("app.env", prevEnv)
		facades.Config().Add("app.debug", prevDebug)
	})
	WarnInsecureDefaults() // logs only
}

func TestCorsAllowsAnyOrigin(t *testing.T) {
	prev := facades.Config().Get("cors.allowed_origins")
	t.Cleanup(func() { facades.Config().Add("cors.allowed_origins", prev) })

	cases := []struct {
		name    string
		origins any
		want    bool
	}{
		{"wildcard", []string{"*"}, true},
		{"empty", []string{}, false},
		{"allowlist", []string{"https://admin.example.com"}, false},
		{"any wildcard", []any{"*"}, true},
		{"any wildcard after blank", []any{"", "*"}, true},
		{"any allowlist", []any{"https://admin.example.com"}, false},
		{"unknown type falls back", "bogus", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			facades.Config().Add("cors.allowed_origins", tc.origins)
			if got := corsAllowsAnyOrigin(); got != tc.want {
				t.Fatalf("corsAllowsAnyOrigin() = %v, want %v", got, tc.want)
			}
		})
	}
}
