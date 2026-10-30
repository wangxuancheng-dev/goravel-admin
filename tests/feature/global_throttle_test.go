package feature_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/goravel/framework/facades"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"goravel/app/http/helpers"
	"goravel/app/utils/errorlog"
	"goravel/tests"
)

// The global limit (1000/min per client IP and path) must not reset when a client rotates
// X-Forwarded-For. Test requests come from 192.0.2.1, which is not a trusted proxy, so the
// header must be ignored and every request lands in the same bucket.
func TestGlobalThrottleIgnoresSpoofedForwardedFor(t *testing.T) {
	key := "throttle:global:0:192.0.2.1:/health"
	facades.Cache().Forget(key)
	t.Cleanup(func() { facades.Cache().Forget(key) })

	testCase := tests.TestCase{}
	get := func(i int) int {
		resp, err := testCase.Http(t).
			WithHeader("X-Forwarded-For", fmt.Sprintf("203.0.%d.%d", i/250, i%250+1)).
			Get("/health")
		require.NoError(t, err)
		if resp.IsSuccessful() {
			return 200
		}
		content, _ := resp.Content()
		t.Logf("request %d refused: %s", i, content)
		return 429
	}

	for i := 0; i < 1000; i++ {
		require.Equal(t, 200, get(i), "request %d is within the limit", i)
	}
	assert.Equal(t, 429, get(1000), "a new X-Forwarded-For must not buy a fresh bucket")
}

// System/error log rows must record the proxy-aware IP; errorlog cannot import helpers,
// so the route provider wires it in at boot.
func TestErrorLogUsesProxyAwareClientIP(t *testing.T) {
	assert.Equal(t,
		reflect.ValueOf(helpers.GetRealIP).Pointer(),
		reflect.ValueOf(errorlog.ClientIP).Pointer())
}
