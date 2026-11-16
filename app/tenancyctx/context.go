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

// detach copies tenant routing keys onto a fresh Background context: the tenant stays,
// the cancellation (and deadline) of from does not. Only Go uses it: background work
// starts through Go, which also recovers panics.
func detach(from context.Context) context.Context {
	bg := context.Background()
	if from == nil {
		return bg
	}
	id, ok := IDFrom(from)
	if !ok {
		return bg
	}
	conn, _ := ConnectionFrom(from)
	code, _ := CodeFrom(from)
	return WithTenant(bg, id, conn, code)
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
