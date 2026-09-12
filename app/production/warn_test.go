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
