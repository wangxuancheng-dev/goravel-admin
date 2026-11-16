# Agent notes (any AI coding tool)

Project-specific hard rules for automated coding agents (Cursor, Claude Code, Copilot, etc.).

## Tenancy (always write as if multi-tenant)

Read: `website/docs/advanced/tenancy-dev.md` (English: `website/docs/en/advanced/tenancy-dev.md`).

| Correct | Incorrect |
|---------|-----------|
| `appfacades.OrmQuery(ctx)` for tenant business data | `facades.Orm().Query()` |
| `PlatformOrmQuery(ctx)` / `PlatformQuery()` for platform models | `OrmQuery` on tenants / `platform_*` / plans |
| `tenancyctx.Go(ctx, name, fn)` for async | `go func` capturing request ctx |
| `tenancy.CacheKey(ctx, key)` / `StoragePrefix(ctx)` | unprefixed cache/storage keys |

Local / CI guard:

```bash
go run ./scripts/tenantlint
```

Do not edit tenant ops / resolution / fleet backup paths unless the user explicitly requested it.

## Backend CRUD

Follow the code generator (`app/services/templates/`) and `article_*`. Cursor skill: `.cursor/skills/goravel-admin-backend/SKILL.md`.

## Comments

New/edited code comments and Go docstrings: English (ASCII). User-facing Chinese only in i18n JSON / website docs.
