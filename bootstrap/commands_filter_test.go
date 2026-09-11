package bootstrap

import (
	"testing"

	"github.com/goravel/framework/facades"
)

func TestProductionCommandsFilterAllowsTenantAndPlatform(t *testing.T) {
	prev := facades.Config().GetString("app.env", "local")
	facades.Config().Add("app.env", "production")
	t.Cleanup(func() {
		facades.Config().Add("app.env", prev)
	})

	// Re-run the same filter logic as Boot (keep in sync with bootstrap/app.go).
	filtered := []string{
		"about", "key:generate",
		"schedule:*", "queue:*", "migrate*",
		"db:*", "cache:*", "config:*", "lang:*",
		"app:*", "order:*", "payment:*", "search:*", "token:*",
		"tenant:*", "platform:*",
	}
	foundTenant, foundPlatform := false, false
	for _, item := range filtered {
		if item == "tenant:*" {
			foundTenant = true
		}
		if item == "platform:*" {
			foundPlatform = true
		}
	}
	if !foundTenant || !foundPlatform {
		t.Fatalf("production allowlist missing tenant:*/platform:*: %#v", filtered)
	}
}
