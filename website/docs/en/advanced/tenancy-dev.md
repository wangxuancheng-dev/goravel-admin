# Multi-tenancy: developer guide

This is the one page to read before writing a business module. Architecture, deployment and operations are in [Multi-tenancy](/en/advanced/tenancy); tuning is in [Scale and recommended settings](/en/advanced/tenancy-scaling).

Everything below is also correct in single-database mode (`TENANCY_DRIVER=off`). **Always write for multi-tenancy**; there is no second way to learn.

## Three rules

### 1. Database: pass the ctx you have

```go
// Tenant business data: the request / job ctx (an http.Context is a context.Context)
appfacades.OrmQuery(ctx).Where("status", 1).Find(&orders)
appfacades.OrmTransaction(ctx, func(tx orm.Query) error { ... })

// Platform data (tenants, platform_*, plans and subscriptions)
appfacades.PlatformOrmQuery(ctx)   // when you have a ctx
appfacades.PlatformQuery()         // no ctx: console commands, schedulers, ops
```

- Business service constructors take a `context.Context` and keep it as `s.ctx`.
- Never write `facades.Orm().Query()`: it does not know which tenant is current.
- Data statements in migrations use `migrationQuery()`.

### 2. Background work: `tenancyctx.Go`

```go
adminID := ...  // read what you need from the http.Context first
tenancyctx.Go(ctx, "audit_export", func(ctx context.Context) {
    appfacades.OrmQuery(ctx).Create(&row) // still the same tenant
})
```

`tenancyctx.Go` does three things:

- It keeps the tenant.
- It drops the request's cancellation (the original ctx is cancelled as soon as the request ends).
- It recovers a panic, so one background task cannot take the process down.

The framework recycles the `http.Context` after the request, so the goroutine must not read it.

### 3. Cache and storage: tenant prefix

```go
facades.Cache().Put(tenancy.CacheKey(ctx, "report:"+id), v, ttl)
path := tenancy.StoragePrefix(ctx) + "exports/" + name
```

A key that skips `tenancy.CacheKey` is shared by every tenant. Building keys in a helper is fine (e.g. `func (s *X) lockKey(...)` that calls `tenancy.CacheKey`); the checker recognises it.

## What happens when you get it wrong

None of this relies on memory. Mistakes show up while you develop:

| Mistake | Result |
|---|---|
| Forgotten or unbound ctx: `OrmQuery(nil)` / `OrmQuery(context.Background())` | With tenancy on: `ErrNoTenantContext`, never a silent platform query |
| Platform table through a tenant ctx: `OrmQuery(ctx).Model(&models.Tenant{})` | Refused by the tenant connection with `tenantdb.ErrPlatformTable`, naming `PlatformQuery()` |
| `facades.Orm().Query()`, `PlatformOrmQuery(nil)`, unprefixed cache keys, request ctx inside `go func` | CI fails in `tenantlint` |
| An endpoint returns tenant A's data to tenant B | `TestEveryAdminRouteIsTenantIsolated` scans every GET route, including new ones |
| Raw SQL reaching another database | With `TENANCY_DB_DEDICATED_USER`, the tenant's database account can only reach its own database, so the server refuses it |

Check locally:

```bash
go run ./scripts/tenantlint
go test ./tests/feature -run 'TestEveryAdminRouteIsTenantIsolated|TestTenantResolutionSecurity|TestTenantConnectionRefusesPlatformTables'
```

## When an exception is intended

Some keys are global on purpose, such as platform alert debounce or OIDC state. Say why at the call site or in the function's doc comment:

```go
//tenantlint:allow cache: platform alert debounce, not tenant data
func AlertQueueBacklogIfNeeded(...) { ... }
```

The rules are `orm`, `nilctx`, `cache` and `goctx`; the reason after the colon is required.

## Adding a platform table

1. Make the model implement `models.PlatformModel` and add it to `platformModels` in `app/models/platform.go`.
2. Start the migration's `Up` / `Down` with `if SkipOnTenantConnection() { return nil }` and register it in `IsLandlordOnlyMigration`.

`TestPlatformTablesMatchLandlordOnlyMigrations` fails when the two disagree.

## Code that needs an experienced reviewer

The checks above cover routine tenant code. These areas involve both the platform and tenants, or cross tenants on purpose; have someone who knows the tenancy layer review them before merging:

- Tenant resolution: `middleware/tenant.go`, `BindHTTP`, subdomain / custom-domain lookup.
- Platform operations: `tenant_ops_*`, backup / restore, fleet batches (they cross tenants by design).
- Shared resources outside the database: search index names, queue payloads, object storage buckets.
- `WithSchemaContext`: only for code that cannot take a ctx (migrations, seeders), never for business code.
