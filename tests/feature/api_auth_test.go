package feature_test

import (
	"strings"
	"testing"

	"github.com/goravel/framework/facades"
	"github.com/stretchr/testify/assert"

	"goravel/tests"
)

func TestHealthEndpoint(t *testing.T) {
	testCase := tests.TestCase{}
	resp, err := testCase.Http(t).Get("/health")
	assert.NoError(t, err)
	resp.AssertOk()
	resp.AssertJson(map[string]any{"status": "healthy"})
}

func TestReadyEndpoint(t *testing.T) {
	prevCache := facades.Config().GetString("cache.default")
	prevQueue := facades.Config().GetString("queue.default")
	facades.Config().Add("cache.default", "memory")
	facades.Config().Add("queue.default", "sync")
	t.Cleanup(func() {
		facades.Config().Add("cache.default", prevCache)
		facades.Config().Add("queue.default", prevQueue)
	})

	testCase := tests.TestCase{}
	resp, err := testCase.Http(t).Get("/ready")
	assert.NoError(t, err)
	resp.AssertOk()
	resp.AssertJson(map[string]any{"status": "ready"})
}

func TestHealthReadyAlias(t *testing.T) {
	prevCache := facades.Config().GetString("cache.default")
	prevQueue := facades.Config().GetString("queue.default")
	facades.Config().Add("cache.default", "memory")
	facades.Config().Add("queue.default", "sync")
	t.Cleanup(func() {
		facades.Config().Add("cache.default", prevCache)
		facades.Config().Add("queue.default", prevQueue)
	})

	testCase := tests.TestCase{}
	resp, err := testCase.Http(t).Get("/health/ready")
	assert.NoError(t, err)
	resp.AssertOk()
	resp.AssertJson(map[string]any{"status": "ready"})
}

func TestAPIUserRegisterValidation(t *testing.T) {
	testCase := tests.TestCase{}
	resp, err := testCase.Http(t).
		WithHeader("Content-Type", "application/json").
		Post("/api/user/register", strings.NewReader(`{}`))
	assert.NoError(t, err)
	resp.AssertBadRequest()
}

func TestAPIUserLoginValidation(t *testing.T) {
	testCase := tests.TestCase{}
	resp, err := testCase.Http(t).
		WithHeader("Content-Type", "application/json").
		Post("/api/user/login", strings.NewReader(`{}`))
	assert.NoError(t, err)
	resp.AssertBadRequest()
}
