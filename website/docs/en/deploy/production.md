# Production checklist

Ops conventions for public deploy. Feature boundaries: [Open-source scope](/en/guide/opensource). Multi-tenancy: [Tenancy](/en/advanced/tenancy). Go-live checklist: [SaaS checklist](/en/advanced/saas).

Full Chinese detail: [生产清单](/deploy/production).

## Before go-live

1. `migrate` succeeds; after `db:seed`, change default `admin` password  
   - Multi-tenant: also `tenant:migrate-all` (concurrency: `TENANCY_MIGRATE_CONCURRENCY` / `--concurrency`; see [Tenancy concurrency knobs](/en/advanced/tenancy-scaling#concurrency-knobs-tune-with-queue-workers))
2. Redis up; web process + queue workers stay running. `CACHE_STORE=redis` and `QUEUE_CONNECTION=redis` are enforced when `APP_ENV=production`: any other driver makes the server refuse to start (artisan commands still run)  
3. HTTPS + reverse proxy  
4. Disable or lock down: Swagger, pprof, code generator  
5. Log disk + backup policy ready  
6. Start from `.env.production.example` — **do not** ship `docker-compose.yml` default passwords to production  
7. Set `PAYMENT_GATEWAYS_ENABLED` explicitly; omit `mock` unless intentionally testing  
8. After upgrades that add Task Center / CSV import: re-seed menus/permissions (next section)

## Minimal tenancy path

When enabling DB-per-tenant in public production, turn on the whole block in `.env.production.example` (“Minimal DB-per-tenant production path”), not only `TENANCY_DRIVER`:

| Setting | Value |
|---------|--------|
| `TENANCY_RESOLVER` | `subdomain` |
| `TENANCY_BASE_DOMAIN` | your public apex |
| `TENANCY_ALLOW_PLATFORM_DB_CREDENTIALS` | `false` |
| `TENANCY_DB_DEDICATED_USER` | `true` |
| `TENANCY_MYSQL_TLS` / `TENANCY_POSTGRES_SSLMODE` | `true` / `require` |
| `CACHE_STORE` / `QUEUE_CONNECTION` | `redis` (+ `queue-long-running` worker) |
| `TRUSTED_FORWARDED_HOST_ONLY` | `true` after `TRUSTED_PROXIES` is correct |

Backfill existing same-host tenants (created before dedicated users):

```bash
go run . artisan tenant:ensure-db-user --all --dry-run
go run . artisan tenant:ensure-db-user --all
# already-dedicated tenants are skipped; add --rotate to change their passwords
# optional: --force replaces a custom same-host username (old DB account is kept)
```

Full Chinese detail: [生产清单 §1.0](/deploy/production#10-最小生产默认路径一户一库).

### You can ignore for mid-size (for now)

Still available, but skip until you need them (~&lt;50 active tenants, no sold plans, no search):

- Schema isolation, sharding, fleet concurrency knobs ([Scale](/en/advanced/tenancy-scaling))
- Meilisearch / Elasticsearch, entitlement enforcement
- Vanity domains, OpenTelemetry, AI Lab

Tier overview: [Open-source scope · Capability tiers](/en/guide/opensource#capability-tiers-keep-features-expose-in-layers).

## Task Center + import permissions

After shipping versions with import/export Task Center, confirm menu + permissions exist (otherwise no sidebar entry or 403 on import APIs).

| Item | Detail |
|------|--------|
| Menu | System → Import/Export; component path **`export/TaskCenter`** (React and Vue) |
| Import slugs | `import.index`, `import.show`, `import.download_error`, `import.destroy` |
| Module import | Generator modules also need `{module}.import` (e.g. `article.import`) on roles |

```bash
# Dev
go run . artisan db:seed --seeder=MenuSeeder
go run . artisan db:seed --seeder=PermissionSeeder

# Production binary (APP_ENV=production requires --force)
./main artisan db:seed --force --seeder=MenuSeeder
./main artisan db:seed --force --seeder=PermissionSeeder
```

`PermissionSeeder` upserts import permissions under the export menu; `MenuSeeder` keeps `Component: export/TaskCenter`. Full `db:seed` is for first-time init only. Non-super roles must be granted the new slugs in Role management. When `APP_ENV=production`, Goravel requires `--force` on `db:seed`.

## Health

- `/health` — liveness  
- `/ready` — readiness (DB / critical deps)

Do **not** put queue backlog scans on `/ready` (latency). Use the scheduled command below.

### Queue backlog alert (offline)

`queue:alert-backlog` (hourly in `app/console/kernel.go`) sums Redis pending via `QueueStatsReader`; POSTs webhook when over threshold (1h cache debounce). Webhook URL: `QUEUE_ALERT_WEBHOOK_URL`, else `READY_ALERT_WEBHOOK_URL`. Threshold: `QUEUE_ALERT_BACKLOG_THRESHOLD` (default `100`).

```bash
./main artisan queue:alert-backlog
```

In **Observability → Queue**, check whether the webhook is configured and use **Send test alert** (`observability.queue_alert_test`). Configure the URL via `.env` (not the UI).

White-label: System Config → Website (name / logo). Login uses `GET /api/admin/login/branding`.

Platform tenant list: filter by provision status, **Retry all failed migrates**, summary at `GET /api/platform/tenants/ops-summary`.

Ready failure webhook: `READY_ALERT_WEBHOOK_URL` (5 min debounce on `/ready` non-200).
## Processes: API vs Queue Worker

Same binary and shared Redis/DB. Split roles via `.env` — do not run full HTTP + all queue runners on every node.

Sizing by active-tenant band (API / Worker / DB hosts): [Tenancy · Scale](/en/advanced/tenancy-scaling).

| Role | Purpose | `APP_DISABLED_RUNNERS` |
|------|---------|--------------------------|
| **API** | HTTP only; dispatch jobs | `queue-*` (disables all `queue-*` runners) |
| **Worker** | Consume queues | leave empty (do **not** disable `queue-*`); needs **`queue-schedule`** for flexible cron |
| **Schedule** | Optional | on API replicas add `goravel:schedule`; run schedule on **one** node only |

**API node:**

```ini
CACHE_STORE=redis
QUEUE_CONNECTION=redis
APP_DISABLED_RUNNERS=queue-*
# APP_DISABLED_RUNNERS=queue-*,goravel:schedule
```

**Worker node:**

```ini
CACHE_STORE=redis
QUEUE_CONNECTION=redis
QUEUE_CONCURRENT=2
QUEUE_LONG_RUNNING_CONCURRENT=1
QUEUE_SCHEDULE_CONCURRENT=10
```

Use `queue-*` (hyphen), not `queue:*`. The latter is for production Artisan command filters, not queue runners. See Chinese [生产清单](/deploy/production) §4.1.

### WebSocket on multiple API nodes

Notification WS hub is process-local; with **`WEBSOCKET_REDIS_BRIDGE=true`** (default), every normal `./main` Boot starts a Redis Pub/Sub subscriber — **no separate WS command**. Any API/Worker that calls `Broadcast` / `SendToAdmin` fans out to clients connected on other API processes (same Redis as `REDIS_*`).

Requirements: reachable Redis (`database.redis.default`). Disable with `WEBSOCKET_REDIS_BRIDGE=false` for single-node / no Redis. Optional sticky sessions on `/ws` still help reconnect affinity but are not required for cross-node push.

**Vanity / custom admin domains** (page apex differs from API apex, e.g. `tenant.example.xyz` → `api.example.top`): the browser cannot complete cross-site WSS to the API host. The SPA uses same-origin `wss://<page-host>/ws/admin/notifications`. Deploy `html-react/worker.js` (or `html/worker.js`) with `[vars] API_ORIGIN=https://api.example.top` so Cloudflare proxies `/ws` to the API. If the SPA is on nginx instead, add a `/ws` `proxy_pass` on that vhost (see root `nginx.conf` comments).

Monitor **WS online admins / WS connections** aggregates cluster-wide from a Redis ZSET (`WEBSOCKET_REDIS_PRESENCE_KEY`, default `goravel:ws:presence`) when the bridge is on; stale entries expire ~45s after disconnect.

## Database versions

| Database | Tested | Notes |
|---|---|---|
| MySQL | 8.4 LTS (CI, local) | 8.0 reached end of life in 2026-04. New accounts use `caching_sha2_password` (supported by the Go driver); before MySQL 9, convert accounts still on `mysql_native_password` (9 removes that plugin) |
| PostgreSQL | 16 (CI), 17 (local); 13+ required | `pg_dump` / `psql` must be the server's major version or newer; tenant databases are dropped with `DROP DATABASE ... WITH (FORCE)` (13+, every supported release) |

- Dialect details are left to the framework where it offers them (gorm's dialector, see `app/utils/db_dialect.go`): identifier quoting, savepoints, duplicate-key detection (`gorm.ErrDuplicatedKey`); table existence and listings come from the framework Schema; driver checks use the driver packages' `Name` constants. Errors are never classified by parsing message text (PostgreSQL translates messages per `lc_messages`). Hand-written SQL remains only where the framework has nothing: creating / dropping tenant databases, backup / restore commands, and PostgreSQL table lookups scoped to one schema (`app/tenantdb/pgschema.go`; the framework's scans the whole database and can deadlock).
- A tenant on another driver than the platform database needs host and port (no fallback to the platform's `DB_HOST` / `DB_PORT`).
- Since PostgreSQL 15 only the database owner may create in `public`: when the platform account creates a tenant that has its own account, the database is created `OWNER <tenant account>` and a schema `AUTHORIZATION <tenant account>`. The platform account needs that right (superuser, or member of the role); otherwise tenant creation fails with the server's reason.
- Data statements in migrations use `migrationQuery()` (inside the migration transaction); PostgreSQL DDL is transactional, so `facades.Orm()` cannot see a table the migration just created.

## Resource ownership (admin)

- **Exports:** download / SSE progress / delete — owner or configured `admin.super_admin_id`  
- **Attachments:** private R/W same rule; public (`is_public=1`) readable by logged-in admins; mutate still owner/super  
- **Payments:** use **mock** for end-to-end; WeChat/Alipay query & notify verify still stub (`payment_gateway_not_implemented`)

## Admin SPA (React) & Docker

**React (`html-react/`) is the primary UI**; Vue (`html/`) is the peer implementation. Images default to `BUILD_FRONTEND=0` (API-only). Set `BUILD_FRONTEND=1` to build **React** into `public/admin`.

```bash
cd html-react && npm ci && npm run build
docker build --build-arg BUILD_FRONTEND=1 -t goravel-admin .
```

Full Chinese detail: [生产清单](/deploy/production) §6.

### Frontend static release: rollback & canary

The admin UI is a Vite static SPA (`html-react/dist` or `html/dist`). There is **no** built-in percentage canary switch; use the release method:

| Method | Rollback | Canary |
|--------|----------|--------|
| Nginx hosting `dist` | Keep previous release dir; flip symlink | Dual dirs / LB weighted split |
| Cloudflare Workers | Dashboard / wrangler previous version | Platform gradual deployments |
| Docker blue/green (`BUILD_FRONTEND=1`) | `scripts/deploy/rollback.sh` | **Full cutover** after health checks, not % traffic |

Server layout:

```text
/var/www/admin/
  current -> releases/20260914_1020
  releases/
    20260914_1000/
    20260914_1020/
```

Ship: unpack `dist` into `releases/<stamp>` → point `current` → `nginx -s reload`.  
Rollback: repoint `current` → reload. Keep the **whole** directory (hashed assets); do not replace only `index.html`.

Percentage canary needs Nginx/LB weights or Cloudflare Gradual Deployments. Repo blue/green scripts cover the **API container** (optional embedded SPA); see [Docker](/en/deploy/docker).

## Related

- Tenant scale / pool starting points: [Tenancy · Scale and recommended settings](/en/advanced/tenancy-scaling)
- [Build & deploy](/en/deploy/build)  
- [Docker production](/en/deploy/docker)  
- [Testing](/en/guide/testing)  
- [Open-source scope](/en/guide/opensource)

## 9. Prometheus, OIDC SSO, audit retention

### Prometheus

```ini
METRICS_ENABLED=true
METRICS_TOKEN=strong-random-token
```

- Scrape `GET /metrics` (optional Bearer `METRICS_TOKEN`)
- Includes Go/process collectors plus `goravel_admin_http_requests_*` from admin API middleware
- Sample `nginx.conf` isolates `/metrics`; use ACL and/or token in production

### OIDC / Enterprise SSO

```ini
OIDC_ENABLED=true
OIDC_ISSUER=https://login.microsoftonline.com/{tenant}/v2.0
OIDC_CLIENT_ID=
OIDC_CLIENT_SECRET=
OIDC_REDIRECT_URL=https://api.example.com/api/admin/auth/oidc/callback
OIDC_FRONTEND_REDIRECT=https://admin.example.com/login/oidc-callback
OIDC_AUTO_PROVISION=false
```

- Login page shows SSO when enabled; callback matches admin by **email** and issues JWT
- Multi-tenant: SSO start URL must include `?tenant_code=` (SPA does this); OIDC `state` embeds `tenant_id` and the callback re-binds the tenant DB (callback route has no Tenant middleware)
- Works with header/query, built-in subdomain, and vanity domains (login branding returns `tenant_code` so vanity SPAs can put it on the redirect URL)
- Default IdP from process `OIDC_*`; each tenant may override Issuer/Client under **System config → Enterprise SSO (OIDC)** (configs group `oidc`). `OIDC_REDIRECT_URL` / `OIDC_FRONTEND_REDIRECT` stay global
- Profile page can list/revoke login sessions (`GET/DELETE /api/admin/auth/tokens`)
- IP allowlist menu: when any enabled row exists, only listed IPs may access (empty = unrestricted). Checked on login and admin APIs
- Audit webhook: `AUDIT_WEBHOOK_URL` or tenant `audit.webhook_url` — async JSON POST for login/operation logs
- `OIDC_AUTO_PROVISION=true` creates a **disabled** admin (manual role assignment); keep off in production
- Requires Cache (Redis recommended) for OIDC `state` (global key, not tenant-prefixed)

### Audit retention

```ini
AUDIT_LOG_RETENTION_DAYS=30
```

- Schedules: `operation_log:archive`, `login_log:archive` (CSV to task center, then delete)
- Export without delete: `POST /api/admin/operation-logs/export?days=30` (`operation_log.export`)
