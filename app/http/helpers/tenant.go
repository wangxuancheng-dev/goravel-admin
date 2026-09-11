package helpers

import (
	"context"
	"errors"

	"github.com/goravel/framework/contracts/http"

	"goravel/app/tenancy"
	"goravel/app/tenancyctx"
)

const (
	ContextKeyTenantID         = tenancyctx.KeyTenantID
	ContextKeyTenantConnection = tenancyctx.KeyTenantConnection
	ContextKeyTenantCode       = tenancyctx.KeyTenantCode
)

// TenancyEnabled is an alias of tenancy.Enabled (single source of truth).
func TenancyEnabled() bool {
	return tenancy.Enabled()
}

// SetTenantContext writes tenant id / connection / code onto the HTTP context.
func SetTenantContext(ctx http.Context, tenantID uint, connectionName, code string) {
	if ctx == nil {
		return
	}
	ctx.WithValue(tenancyctx.KeyTenantID, tenantID)
	ctx.WithValue(tenancyctx.KeyTenantConnection, connectionName)
	if code != "" {
		ctx.WithValue(tenancyctx.KeyTenantCode, code)
	}
}

func GetTenantConnectionFromContext(ctx context.Context) (string, bool) {
	return tenancyctx.ConnectionFrom(ctx)
}

func GetTenantIDFromContext(ctx http.Context) (uint, bool) {
	if ctx == nil {
		return 0, false
	}
	return tenancyctx.IDFrom(ctx)
}

func GetTenantIDFromAnyContext(ctx context.Context) (uint, bool) {
	return tenancyctx.IDFrom(ctx)
}

func TenantCacheKey(ctx context.Context, key string) string {
	return tenancy.CacheKey(ctx, key)
}

func TenantStoragePrefix(ctx context.Context) string {
	return tenancy.StoragePrefix(ctx)
}

// TenantBound reports whether the request/job ctx is bound to a tenant connection.
func TenantBound(ctx context.Context) bool {
	return tenancy.Bound(ctx)
}

func RequireTenant(ctx http.Context) (uint, error) {
	tenantID, ok := GetTenantIDFromContext(ctx)
	if !ok || tenantID == 0 {
		return 0, errors.New("tenant_id is required")
	}
	return tenantID, nil
}
