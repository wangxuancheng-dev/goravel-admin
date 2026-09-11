package models

// Historical note: row-level TenantScope / ScopeTenant / GlobalScopes(tenant_id)
// are intentionally not used. Isolation is database-per-tenant (or PG schema)
// via TENANCY_DRIVER=database + OrmQuery(ctx). See docs/TENANT_RESERVED.md.
