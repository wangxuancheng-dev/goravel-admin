package helpers

import (
	"errors"

	"github.com/goravel/framework/contracts/http"

	"goravel/app/tenancyctx"
)

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

func RequireTenant(ctx http.Context) (uint, error) {
	tenantID, ok := tenancyctx.IDFrom(ctx)
	if !ok || tenantID == 0 {
		return 0, errors.New("tenant_id is required")
	}
	return tenantID, nil
}
