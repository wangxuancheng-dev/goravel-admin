package middleware

import (
	"testing"

	"github.com/goravel/framework/contracts/http"
	"github.com/stretchr/testify/assert"
)

func TestIsImpersonationBlocked(t *testing.T) {
	blocked := []struct{ method, path string }{
		{http.MethodPut, "/api/admin/password"},
		{http.MethodPost, "/api/admin/google-authenticator/bind"},
		{http.MethodPost, "/api/admin/google-authenticator/unbind"},
		{http.MethodDelete, "/api/admin/auth/tokens/others"},
		{http.MethodDelete, "/api/admin/auth/tokens/12"},
	}
	for _, c := range blocked {
		assert.True(t, isImpersonationBlocked(c.method, c.path), c.path)
	}

	allowed := []struct{ method, path string }{
		{http.MethodGet, "/api/admin/info"},
		{http.MethodGet, "/api/admin/google-authenticator/status"},
		{http.MethodGet, "/api/admin/auth/tokens/list"},
		{http.MethodPost, "/api/admin/logout"},
		// Managing *other* admins stays allowed (normal permission checks apply).
		{http.MethodPut, "/api/admin/admins/3/password"},
		{http.MethodPost, "/api/admin/admins/3/reset-google-auth"},
	}
	for _, c := range allowed {
		assert.False(t, isImpersonationBlocked(c.method, c.path), c.path)
	}
}
