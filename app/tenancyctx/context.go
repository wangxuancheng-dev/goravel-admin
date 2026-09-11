package tenancyctx

import "context"

// WithTenant returns a context carrying tenant id and ORM connection name (for jobs/workers).
func WithTenant(ctx context.Context, tenantID uint, connectionName, code string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx = context.WithValue(ctx, KeyTenantID, tenantID)
	ctx = context.WithValue(ctx, KeyTenantConnection, connectionName)
	if code != "" {
		ctx = context.WithValue(ctx, KeyTenantCode, code)
	}
	return ctx
}

// CodeFrom returns tenant code if present.
func CodeFrom(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	v := ctx.Value(KeyTenantCode)
	s, ok := v.(string)
	if !ok || s == "" {
		return "", false
	}
	return s, true
}
