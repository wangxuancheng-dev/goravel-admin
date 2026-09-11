package tenancy

import (
	"context"
	"fmt"
	"strings"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"

	"goravel/app/tenancyctx"
)

// Enabled reports whether database-per-tenant mode is on (TENANCY_DRIVER=database).
func Enabled() bool {
	return strings.EqualFold(facades.Config().GetString("tenancy.driver", "off"), "database")
}

// Bound reports whether ctx already carries a tenant ORM connection.
func Bound(ctx context.Context) bool {
	_, ok := tenancyctx.ConnectionFrom(ctx)
	return ok
}

// CacheKey prefixes cache keys with tenant id when tenancy is on and bound.
func CacheKey(ctx context.Context, key string) string {
	if !Enabled() {
		return key
	}
	if id, ok := tenancyctx.IDFrom(ctx); ok {
		return fmt.Sprintf("t%d:%s", id, key)
	}
	return key
}

// StoragePrefix returns object-storage path prefix, e.g. tenants/acme/
func StoragePrefix(ctx context.Context) string {
	if !Enabled() {
		return ""
	}
	if code, ok := tenancyctx.CodeFrom(ctx); ok {
		return fmt.Sprintf("tenants/%s/", code)
	}
	if id, ok := tenancyctx.IDFrom(ctx); ok {
		return fmt.Sprintf("tenants/%d/", id)
	}
	return ""
}

// HTTPHint reads tenant id/code from configured header or query params.
func HTTPHint(ctx http.Context) string {
	if ctx == nil {
		return ""
	}
	headerName := facades.Config().GetString("tenancy.header", "X-Tenant-ID")
	raw := strings.TrimSpace(ctx.Request().Header(headerName, ""))
	if raw == "" {
		raw = strings.TrimSpace(ctx.Request().Query("tenant_id", ""))
	}
	if raw == "" {
		raw = strings.TrimSpace(ctx.Request().Query("tenant_code", ""))
	}
	return raw
}
