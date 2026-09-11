package tenancyctx

import "context"

// Context keys for database-per-tenant routing (shared by middleware and OrmQuery).
const (
	KeyTenantID         = "tenant_id"
	KeyTenantConnection = "tenant_connection"
	KeyTenantCode       = "tenant_code"
)

// ConnectionFrom returns the tenant ORM connection name from ctx, if set.
func ConnectionFrom(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	v := ctx.Value(KeyTenantConnection)
	if v == nil {
		return "", false
	}
	name, ok := v.(string)
	if !ok || name == "" {
		return "", false
	}
	return name, true
}

// IDFrom returns tenant id from any context.Context.
func IDFrom(ctx context.Context) (uint, bool) {
	if ctx == nil {
		return 0, false
	}
	return idFromValue(ctx.Value(KeyTenantID))
}

func idFromValue(v any) (uint, bool) {
	if v == nil {
		return 0, false
	}
	switch id := v.(type) {
	case uint:
		return id, id > 0
	case uint8:
		return uint(id), id > 0
	case uint16:
		return uint(id), id > 0
	case uint32:
		return uint(id), id > 0
	case uint64:
		return uint(id), id > 0
	case int:
		if id > 0 {
			return uint(id), true
		}
	case int64:
		if id > 0 {
			return uint(id), true
		}
	}
	return 0, false
}
