# Multi-tenancy

> Chinese remains the full reference: [多租户](/advanced/tenancy). Key local Docker steps are below in English.
>
> Writing a business module? Read [Multi-tenancy: developer guide](/en/advanced/tenancy-dev). Production tuning: [Scale and recommended settings](/en/advanced/tenancy-scaling).

## Docker local setup

Default `docker compose` / [quick start](/en/guide/getting-started) is **single-DB** (`TENANCY_DRIVER=off`). To try database-per-tenant locally:

1. Bring the stack up per the quick start (`.env` from `.env.docker.example`).
2. Add or change in the root `.env`:

```ini
TENANCY_DRIVER=database
TENANCY_RESOLVER=header
TENANCY_HEADER=X-Tenant-ID
# Same MySQL container may reuse platform DB credentials (local only)
TENANCY_ALLOW_PLATFORM_DB_CREDENTIALS=true

PLATFORM_ADMIN_USERNAME=admin
PLATFORM_ADMIN_PASSWORD=secret
PLATFORM_ADMIN_NAME=Platform Admin
```

3. Restart the API container:

```bash
docker compose up -d app
# or
docker compose restart app
```

4. Bootstrap platform + sample tenant (binary path inside the container is `/www/main`):

```bash
docker compose exec app /www/main artisan platform:install
docker compose exec app /www/main artisan tenant:create acme "Acme" --migrate
```

5. Enable tenant hints for local frontends (`html/.env` or `html-react/.env`):

```ini
VITE_TENANCY_ENABLED=true
VITE_TENANCY_DRIVER=database
VITE_TENANCY_HEADER=X-Tenant-ID
```

6. Open:

| Entry | Notes |
|-------|--------|
| `/platform/login` | Platform console (`PLATFORM_ADMIN_*`) |
| `/login` + header `X-Tenant-ID: acme`, or `/login?tenant_code=acme` | Tenant admin (seed credentials in the tenant DB) |

Notes:

- For public deploy use `TENANCY_RESOLVER=subdomain` and keep `TENANCY_ALLOW_PLATFORM_DB_CREDENTIALS=false`.
- Compose defaults stay single-DB.
- Full commands and production rules: [Chinese tenancy page](/advanced/tenancy).

---

默认 **`TENANCY_DRIVER=off`**：整站单库。

设为 **`database`** 后：每租户独立 database（MySQL）或 database/schema（PostgreSQL）；**租户认证与业务都在该租户库**。平台控制台使用独立账号与 `/api/platform`，不切租户库。

## 架构

```text
TENANCY_DRIVER=database
  ├── 平台库 DB_*
  │     ├── tenants
  │     ├── platform_admins
  │     └── personal_access_tokens（platform_admin）
  └── 租户库
        ├── admins / RBAC / 业务
        └── personal_access_tokens（admin）
```

| | 租户后台 | 平台控制台 |
|--|----------|------------|
| 入口 | `/login` | `/platform/login` |
| API | `/api/admin` | `/api/platform` |
| Token | `token` | `platform_token` |
| 开户 migrate | — | Platform UI async queue / CLI；HTTP create still forbids sync migrate |
| 开户状态 | — | `provision_status`: `pending` → `migrating` → `ready`/`failed` |

### Connection management (`app/tenantdb`)

- Tenant connections are owned by the app's registry `tenantdb.Default`: each one is built from the framework's exported constructors (driver → gorm → Orm) with its own config overlay. **Nothing is written to the global config and nothing enters the framework's connection caches**, so the app decides when a pool is created, replaced and closed.
- Looking up a registered connection on the request path takes only a read lock. Building or rebuilding one (including the ping, 5s timeout) happens outside any lock and is deduplicated per name, so one slow or unreachable tenant database never stalls other tenants.
- `OrmQuery(ctx)` picks the connection named in ctx; an evicted connection is rebuilt from the `tenants` table on its next use.
- Every tenant request reads the `tenants` row (resolution, connection, storage), so rows are cached in the shared cache (`app/tenantcache`; Redis in production; `TENANCY_TENANT_CACHE_TTL`, default 60s). Invalidation is **automatic**: any successful write to the `tenants` table on the platform connection (`Model(t).Update`, `Save`, `Delete`, bulk `Where(...).Delete`, even an in-app `Exec("UPDATE tenants ...")`) **invalidates on every node at once** — only that tenant when the write identifies it, every tenant otherwise. Disabling, maintenance and deletion are never delayed by the cache, and no code has to call an invalidation function. Only edits made outside the app (e.g. a mysql client) may lag by up to one TTL. If the invalidation hooks are not installed, the cache stays off. Each process also keeps a local copy keyed by the versioned key, so a warm lookup reads only the two version counters from Redis; a version change makes the local copy unreachable, so it never serves stale data. **Multi-process deployments (separate API and worker processes, several machines) need a shared cache (`CACHE_STORE=redis`)**: the `memory` store only invalidates within its own process, so other processes may lag by up to one TTL.
- Migrations and seeders can only reach the database through `facades.Schema()` / `facades.Orm()` (they take no arguments), so migrate / seed / `WithSchemaContext(ctx, fn)` bind a dedicated tenant Schema to the **current goroutine** (`WithBoundTenantSchema`). No global state changes and other requests are unaffected. Background goroutines are started with `tenancyctx.Go(ctx, name, fn)` (keeps the tenant, drops the request cancellation, recovers panics) and use `OrmQuery(ctx)`.

## Tenant entitlements (developer)

Plan features are separate from RBAC. Super-admin does **not** bypass entitlements.

### Backend

```go
// Whole module
router.Middleware(middleware.EntitlementModule("guestbook")).Group(...)

// In-module capability (platform UI: Register capability -> module.member.export)
router.Middleware(middleware.EntitlementFeature("module.member.export")).Get("members/export", ...)

// Optional row quota on Create
services.ConsumeModuleRowsFromHTTP(ctx, "guestbook", 1)
```

### Frontend

`GET /api/admin/info` includes `entitlements.features` / `entitlements.limits` when tenancy is on.

- React: `useUserStore((s) => s.hasEntitlement('module.member.export'))`
- Vue: `useUserStore().hasEntitlement('module.member.export')`

When tenancy is off, `hasEntitlement` returns `true`. Capability keys have no menu — hide buttons in the UI yourself.

### Tenant-side: My Plan

Avatar menu in the tenant admin -> **My Plan** (`/subscription`, shown only when `TENANCY_DRIVER=database`). Read-only: current plan, start/expiry (warning within 7 days), quota usage (`quota.*`), and the feature list (enabled / "upgrade to unlock").

API: `GET /api/admin/entitlements/me` (any logged-in admin; no platform-only override reasons).

Blocked features return 403 `tenant_feature_disabled`; quota overruns return 422 `entitlement_limit_exceeded`. Both show a localized "contact your administrator to upgrade" message.

## 配置

```ini
TENANCY_DRIVER=database
TENANCY_RESOLVER=subdomain            # 公网推荐；本地可用 header
TENANCY_HEADER=X-Tenant-ID
TENANCY_ALLOW_HEADER_FALLBACK=        # 空=subdomain 禁止客户端回落
TENANCY_SUBDOMAIN_RESERVED=www,api,admin,platform,static,assets
TENANCY_DATABASE_PREFIX=tenant_
TENANCY_SCHEMA_PREFIX=tenant_
TENANCY_PLATFORM_CONNECTION=       # 可选；钉死平台连接名，默认取 database.default
TENANCY_ALLOW_PLATFORM_DB_CREDENTIALS=false  # 公网默认 false；同机开发可 true
TENANCY_DB_DEDICATED_USER=false           # true: auto dedicated DB account on create; backfill with tenant:ensure-db-user --all
TENANCY_DB_USER_HOST=%                    # host part of the MySQL account; tighten when app and DB are on different hosts, e.g. 10.0.%
                                          # 远程 host 始终要求独立 username/password
TENANCY_POSTGRES_SSLMODE=                 # 空则回落 DB_SSLMODE / disable
TENANCY_MYSQL_TLS=                       # empty=plain TCP; true|skip-verify for remote MySQL
TENANT_BACKUP_KEEP=10                     # tenant:backup keep count; 0=never prune (old name TENANCY_BACKUP_KEEP still honoured)
TENANT_BACKUP_DISK=                       # disk for backups (s3/oss/cos/minio); empty = the host that ran them
TENANCY_MIGRATE_CONCURRENCY=2             # tenant:migrate-all / seed-all parallelism; --concurrency overrides; max 100
TENANCY_POOL_MAX_IDLE_CONNS=10
TENANCY_POOL_MAX_OPEN_CONNS=20

PLATFORM_ADMIN_USERNAME=admin
PLATFORM_ADMIN_PASSWORD=secret
PLATFORM_ADMIN_NAME=平台管理员

# header 解析租户时，浏览器跨域需放行（默认已含）：
# CORS_ALLOWED_HEADERS=...,X-Tenant-ID
```
前端：`VITE_TENANCY_ENABLED=true`（或 `VITE_TENANCY_DRIVER=database`）。

## 生产要点

1. **Pinned platform connection**: `PlatformOrmQuery` uses `tenancy.platform_connection`; tenant migrate / seed bind only the current goroutine and never change `database.default` or other global state.
2. **生产 Artisan**：`APP_ENV=production` 白名单含 `tenant:*` / `platform:*`（开户/迁移可用）。
3. **Connection lifecycle**: `Forget` closes the tenant's pool immediately; connections evicted as idle (`TENANCY_REGISTERED_IDLE_TTL`) or over the cap (`TENANCY_REGISTERED_MAX`) are closed after a 2-minute grace period and rebuilt on their next use.
4. **开户状态**：HTTP/CLI 创建后为 `pending`；平台 UI 异步迁移或 CLI `tenant:migrate` → `migrating` → `ready`/`failed`；未 ready 禁止业务绑定。UI 入队后若 worker 未消费，约 30 分钟后允许重试。
5. **Account isolation**: remote tenant databases always need their own credentials; same-host tenants may share the platform account only with `TENANCY_ALLOW_PLATFORM_DB_CREDENTIALS=true` (**false by default**). The setting is checked on every connection: turning it off stops tenants without credentials of their own from using the platform account at once (`tenant_credentials_required`), backups included. Recommended: `TENANCY_DB_DEDICATED_USER=true`. Provisioning then creates a `t_<code>` account with the platform account (random password, sealed with APP_KEY), granted only that tenant's database (MySQL `GRANT ALL ON tenant_x.*`; on PostgreSQL the role owns the database or schema), and drops it with the storage. Existing same-host tenants: `go run . artisan tenant:ensure-db-user --all` (try `--dry-run` first; already-dedicated accounts are skipped unless `--rotate`). Even a coding mistake cannot cross databases: the server refuses it. The platform account needs `CREATE USER` + `GRANT OPTION` (MySQL) or `CREATEROLE` (PostgreSQL); with PostgreSQL schema isolation in a shared database, also `REVOKE ALL ON SCHEMA public FROM PUBLIC` on the platform database.
6. **Async ops**: per-tenant migrate / seed / backup / restore via `tenant_ops` (`long-running`); production needs Redis + long-running worker. `migrate-all` / `backup-all` remain CLI: `backup-all` enqueues long-running (paged fan-out); scheduled `backup-scheduled` rotates `TENANT_BACKUP_SCHEDULE_BATCH` tenants/day.
7. **Fleet migrate/seed concurrency**: `TENANCY_MIGRATE_CONCURRENCY` (default 2) limits CLI `tenant:migrate-all` / `seed-all` **and** platform UI batch migrate/seed (`tenant_ops_fleet`); CLI `--concurrency=N` overrides. Separate from `QUEUE_LONG_RUNNING_CONCURRENT` (per-tenant / backup) and `QUEUE_SCHEDULE_CONCURRENT` (collection); see [Concurrency knobs](/en/advanced/tenancy-scaling#concurrency-knobs-tune-with-queue-workers).
8. **Daily full-fleet backup**: with `TENANT_BACKUP_SCHEDULE_MODE=full` (default), `tenant:backup-scheduled` enqueues **all** ready tenants; dumps run via `tenant_ops_fleet` + `TENANCY_BACKUP_CONCURRENCY` (default 2, max 50). `MODE=rotate` keeps `TENANT_BACKUP_SCHEDULE_BATCH` paging. CLI `backup-all` / UI batch backup use the same path. Wall clock ≈ ceil(tenants/concurrency)×per-tenant dump seconds; keep `QUEUE_LONG_RUNNING_CONCURRENT=1` (fleet uses one slot).




8. **CLI op logs**: 	enant:migrate / migrate-all / seed / seed-all write 	enant_op_logs (operator=cli, shared atch_id for fleets). Platform can filter last_op / last_op_status and retry failed seeds. Session DB checks use MySQL DATABASE() and PostgreSQL current_database().

## 公网部署（推荐）

1. **`TENANCY_RESOLVER=subdomain`**: tenants are reached as `acme.example.com`; the apex, `www.` and `platform.` **refuse** a tenant given by Header/Query (unless `TENANCY_ALLOW_HEADER_FALLBACK=true`). API hosts such as `api.`, IPs and other hosts still accept one (split SPA / API, public `<img ?tenant_code=>`); choosing a tenant is not access to it, since tokens are only valid in that tenant's own database.
2. 子域与 Header/body 冲突 → `tenant_hint_conflict`（400）。
3. **支付回调**：`POST /api/payment/notify/{type}/{tenant_code}`（渠道不会带租户 Header）；tenancy 开启时须带 `{tenant_code}`。
4. **Connection pool**: per-tenant `TENANCY_POOL_MAX_*` (default idle 10 / open 20); at higher tenant counts, tighten using [Scale and recommended settings](/en/advanced/tenancy-scaling) so MySQL is not exhausted.
5. 平台控制台走 `platform.` 或独立域名；勿与租户子域混用。
6. **`TENANCY_DB_DEDICATED_USER=true`**: one DB account per tenant that can only reach its own database, so even a coding mistake cannot cross databases (see production note 5). New tenants get an account automatically; backfill existing same-host tenants with `tenant:ensure-db-user --all` (see [Minimal tenancy path](/en/deploy/production#minimal-tenancy-path)).



## Custom domains (`tenant_domains`)

Subdomain `{code}.${TENANCY_BASE_DOMAIN}` works by default. Bind vanity hosts in the **platform console** — no per-tenant Nginx edits, API restarts, or `CORS_ALLOWED_ORIGINS` entries.

| Case | Approach |
|------|----------|
| Most tenants | Wildcard `*.example.com` + subdomain |
| Vanity host, no CDN | `ssl_mode=edge`: CNAME to `TENANCY_DOMAIN_TARGET`; edge on-demand TLS |
| Customer CDN + SSL | `ssl_mode=customer_cdn`: CDN origin to shared ingress; **keep Host as vanity domain** |
| Cloudflare Worker / Pages | `customer_cdn`: add Custom Domain on the same app; DNS needs **TXT only** for ownership. Do not orange-cloud CNAME to `TENANCY_DOMAIN_TARGET` (often 522) |

Resolve order: `active` vanity Host → Origin host (split SPA/API) → subdomain → Header/Query.

Set `TENANCY_BASE_DOMAIN` when using vanity hosts so subdomain resolution only matches `{code}.that-apex` (avoids treating `crm.customer.com` as tenant code `crm`). C-end `/api/user` and `/api/public/*` share the same Host binding. Frontend builds should set `VITE_TENANCY_BASE_DOMAIN` to the same apex so subdomain login can hide the tenant code field.

### CORS (`CORS_ALLOWED_ORIGINS`)

| Setting | Effect |
|------|------|
| Empty / no valid entries / `*` | Allows **any Origin** (handy for demos; avoid as a long-term production lockdown) |
| Explicit allowlist | Auto-appends `http(s)://*.{TENANCY_BASE_DOMAIN}`; **do not** list each tenant subdomain |
| Vanity host + split API | Prefer same-Host reverse proxy for `/api` (no CORS); demos may use empty/`*`; or temporarily add that Origin |

App CORS middleware also tries to allow Origins whose host is an **active** row in `tenant_domains`. If preflight still fails (e.g. framework `rs/cors` only sees the static list), use empty/`*` or same-Host — **do not** edit env for every new merchant domain. Include `X-Tenant-ID` in `CORS_ALLOWED_HEADERS` when resolving tenants via header.

### Origin guard (`ADMIN_ORIGIN_GUARD`)

Only when multi-tenancy is enabled (`TENANCY_DRIVER` not `off`). On split deploys (e.g. Pages SPA + `api.*`), `CORS_ALLOWED_ORIGINS` is often empty/`*` for preflight. Set `ADMIN_ORIGIN_GUARD=true` so `/api/admin` checks the browser `Origin` host:

| Source | Allowed |
|------|----------|
| `DOMAINS_ADMIN` (incl. `*.` wildcards) | yes |
| `TENANCY_BASE_DOMAIN` and its subdomains | yes |
| Active `tenant_domains` vanity hosts | yes |
| Missing `Origin` (curl / native) | yes |
| Anything else | no (403 `origin_not_allowed`) |

```ini
ADMIN_ORIGIN_GUARD=true
DOMAINS_ADMIN=admin.example.com,platform.example.com
TENANCY_BASE_DOMAIN=example.com
```

Default `false`; ignored when tenancy is off. Pairs with Host middleware `DomainAllowTenantVanity`: Host is where the request lands; Origin is where the SPA came from.


```ini
TENANCY_BASE_DOMAIN=example.com
TENANCY_DOMAIN_TARGET=tenants.example.com
TENANCY_DOMAIN_VERIFY_PREFIX=_goravel-tenant
TENANCY_DOMAIN_CACHE_TTL=60
```

### Ops: `TENANCY_DOMAIN_TARGET` (one static ingress)

This is a **single fixed hostname**, not a list of customer domains, and it is unrelated to your platform console host (e.g. `admin.example.com`).

| Role | What to do |
|------|------------|
| You (operator) | Point `tenants.example.com` with **A/AAAA (or your cloud's record)** at the edge / reverse proxy / API ingress that forwards by request **Host (vanity domain)**; for on-demand TLS call `GET /api/platform/public/tls-allow?host=` |
| Merchant | Bind the vanity host in the platform console and pass TXT verify; with `ssl_mode=edge`, **CNAME the customer host to `TENANCY_DOMAIN_TARGET`** (every merchant shares this same target) |
| App | Resolves tenant from `tenant_domains` (`status=active`); **do not** add each customer domain to env |

Example DNS:

```text
# Your static ingress (configure once)
tenants.example.com.     A      203.0.113.10

# Each merchant (many records, same target)
crm.customer-a.com.      CNAME  tenants.example.com.
shop.customer-b.com.     CNAME  tenants.example.com.
```

If you only use subdomains, or all vanity hosts use `customer_cdn`, you may leave `TENANCY_DOMAIN_TARGET` empty. Once you enable edge vanity hosts, that hostname must resolve and accept traffic.

Platform API: `GET/POST /api/platform/tenants/{id}/domains`, `POST .../verify`, `PUT .../primary|disable`, `DELETE`; edge ask `GET /api/platform/public/tls-allow?host=` (HTTP 200 only when edge+active).

Legacy Nginx Host rewrite to `{code}.apex` still works; prefer `tenant_domains` for new setups.

## Scale and recommended settings

Moved to [Multi-tenancy: scale and recommended settings](/en/advanced/tenancy-scaling) (default capacity, dual mode, concurrency knobs, machine layout, example configs, regional cells).

## Object storage and quotas

- **Default shared disk**: one platform `FILESYSTEM_DISK`; paths isolated with `tenants/{code}/`. Tenant admin **cannot** change `file_disk` (save rejected); export format remains editable.
- **Optional BYOB (platform)**: set tenant `storage_mode=custom` with s3/oss/cos/minio credentials (secret sealed with `APP_KEY`). New uploads use disk `tenant_byob_{id}`; attachments store that disk name.
  - Each tenant's storage driver is built from its own credentials and reused within the process, dropped when idle past `TENANCY_REGISTERED_IDLE_TTL` or, over `TENANCY_REGISTERED_MAX`, least recently used first (the same settings as database connections), and rebuilt on next use; the credentials are **never written to the global config**, so no other tenant or code can read them. Credentials changed on the platform take effect on the next use, without a restart.
  - In code, open a disk with `utils.StorageDisk(disk)`; to read a disk's settings (e.g. streaming uploads) use `tenantstorage.DiskConfig(disk)`. Do not read `filesystems.disks.tenant_byob_*` from `facades.Config()` (it is not there).
- **Switching**: shared ↔ custom only changes **new** writes. Old objects remain on the disk recorded in `attachments.disk`. Keep BYOB credentials after switching back to shared so historical custom files stay readable. Purge cleans platform prefix and, if credentials exist, the BYOB prefix too.
- **Delete tenant**: soft-delete into recycle bin; `drop_database` drops DB/schema only; **object storage is not purged here**.
- **Recycle / force delete**: `GET /tenants?trashed=only`; `POST .../undelete`; `POST .../purge` optional retry; `DELETE .../force` defaults to an async purge then hard-delete (frees `code`). A permanent delete (scheduled `tenant:cleanup-deleted` included) always removes search indexes, object storage, backups, the database / schema and the dedicated DB account; if any step fails the tenant stays in the recycle bin with its code, and can be retried. Retention `TENANCY_DELETED_RETENTION_DAYS`; schedule `tenant:cleanup-deleted`.
- **Storage limit** `storage_limit_bytes` (0=unlimited): enforced from `SUM(attachments.size)` before upload.

## 运维增强

1. **Landlord 迁移跳过**：平台表迁移（`tenants` / `platform_admins` / `jobs` / provision/migrate meta）在 `tenant_*` 连接上 `SkipOnTenantConnection` 空跑，避免污染租户库。
2. **Migrate / ops visibility**：`last_migrate_error` / `migrated_at`；platform UI also shows `last_op*` / `last_backup_path`. Seed failure does not demote `ready` → `failed`.
3. **Ping / async ops**：`POST .../ping`；`.../migrate|seed|backup` enqueue `tenant_ops`.
4. **登录限流**：`login` limiter 键含 body/query/header/`subdomain` 租户提示，避免跨租户互相锁号。
5. **日志**：带 `tenant_code` / `tenant_id` 前缀（`app/utils/logger`）。
6. **DB TLS**: `TENANCY_POSTGRES_SSLMODE` / `DB_SSLMODE` for PostgreSQL; `TENANCY_MYSQL_TLS` (`true` / `skip-verify`) for tenant MySQL. `X-Forwarded-Host` is taken from any peer by default (as before); once your proxies / CDN egress are in `TRUSTED_PROXIES`, set `TRUSTED_FORWARDED_HOST_ONLY=true` so only they can set it and a client cannot pick a tenant host by forging it (tokens are still checked per tenant).
   - **Cross-driver tenants**: when the tenant driver differs from the platform database (e.g. MySQL platform, PostgreSQL tenant), host/port/credentials are required; an empty host (meaning "on the platform server") is rejected.
   - **PG schema isolation**: tenants sharing a database each own a schema; table existence checks and table listings only look at the tenant's schema (the framework's own implementation scans every schema in the database), so they migrate safely in parallel.
   - Tests: with `TENANCY_TEST_PG_HOST` / `_PORT` / `_USER` / `_PASSWORD` set, `TestPostgresTenantLifecycle` creates PG tenants (database and schema isolation) on the platform and runs the full lifecycle; CI provides a postgres service.
7. **Backup / restore**: `tenant:backup [--keep=N]`, `tenant:backup-all` (enqueue, not sync dump-all), `tenant:backup-scheduled` (daily rotate).
   - Location: by default `storage/backups/tenants/{code}/` on the host that ran the backup. With several instances set `TENANT_BACKUP_DISK` (a disk of `config/filesystems.go`, e.g. s3 / oss / cos / minio): each dump is uploaded to `tenant-backups/{code}/` on that disk and removed locally, and listing, download, restore, pruning and tenant purge all use the disk, so every instance sees the same files. Alternatively leave it empty and mount `storage/backups` on a shared volume.
   - The framework disk drivers hold a whole file in memory while uploading / downloading; for very large tenant databases prefer the shared volume. In production, a missing or incomplete setting is reported at startup with a `[production]` warning.
   - PG dumps only the tenant's own schema (`pg_dump -n`; `public` for a database-isolated tenant), never other schema tenants in the same database.
   - PG restore empties that schema (`DROP SCHEMA ... CASCADE`, including shards created after the backup) and replays the dump in one transaction: a failure rolls back and leaves the tenant as it was. The platform UI and `tenant:restore` share this implementation.
   - Needs `pg_dump` / `psql` of the server's major version or newer.
8. **CLI scope**: `RunTenantScope` walks **active + ready** only; `tenant:migrate-all` can still cover pending (separate query); concurrency via `TENANCY_MIGRATE_CONCURRENCY` / `--concurrency`.
9. **未绑定隔离**：tenancy 开启但 ctx 未绑定时，`CacheKey` → `t_unbound:*`，`StoragePrefix` → `tenants/_unbound_/`（不与共享根冲突）。

## 首启（推荐）

```bash
# .env 中 TENANCY_DRIVER=database，并配置 APP_KEY、DB_*

go run . artisan platform:install -u admin -p 'your-password'
# 或依赖 .env 的 PLATFORM_ADMIN_*：
# go run . artisan platform:install

# 同机开户 + migrate + seed
go run . artisan tenant:create acme "Acme" --migrate

# 远程库已存在
go run . artisan tenant:create remote "Remote" \
  --host=10.0.0.8 --port=3306 --username=u --password=secret \
  --database=tenant_remote --skip-create --migrate

# 前端
# /platform/login → 平台管理租户
# /login?tenant_code=acme → 租户后台
```

## 命令

```bash
go run . artisan platform:install [-u] [-p] [--name=]
go run . artisan platform:admin {username} {password} [--name=] [--role=owner|viewer]

# role=owner default (full write); viewer = read-only (list/detail/logs/backup download/change own password)

go run . artisan tenant:create {code} {name} \
  [--driver=mysql|postgres] [--isolation=database|schema] \
  [--host=] [--port=] [--username=] [--password=] \
  [--database=] [--schema=] [--skip-create] [--migrate]

go run . artisan tenant:migrate {id|code}
go run . artisan tenant:migrate-all [--concurrency=N]
go run . artisan tenant:seed {id|code} [--class=...]
go run . artisan tenant:seed-all
go run . artisan tenant:list
go run . artisan tenant:enable|disable {id|code}
go run . artisan tenant:backup {id|code} [--keep=N]
go run . artisan tenant:backup-all [--keep=N] [--limit=N] [--rotate]
go run . artisan tenant:backup-scheduled   # needs TENANT_BACKUP_SCHEDULE_ENABLED
go run . artisan tenant:restore {id|code} {sql路径}

# tenancy 开启时，以下命令默认遍历启用租户；可用 --tenant={code|id} 限定
go run . artisan order:create-sharding-tables [--tenant=]
go run . artisan payment:create-sharding-tables [--tenant=]
go run . artisan search:init-orders-index [--tenant=]
go run . artisan search:sync-orders [--tenant=]
go run . artisan search:retry-outbox [--tenant=]
go run . artisan app:clear-logs [--tenant=]
go run . artisan app:clear-chunks [--tenant=]
go run . artisan db:analyze-stats [--tenant=]
go run . artisan db:optimize-tables {tables...} [--tenant=]
# 写多数据命令：tenancy 开启时必须指定 --tenant
go run . artisan order:generate-test-data --tenant={code} --count=1000
go run . artisan payment:generate-test-data --tenant={code} --count=1000
```

### Orphaned databases (run once after upgrading)

Permanent deletes used to remove only the tenant row: the database and dedicated account stayed on the server, and a new tenant with the same code is now refused with `tenant_storage_exists`. Find and remove them:

```bash
go run . artisan tenant:orphans          # list only: prefixed databases / schemas no tenant (recycle bin included) references
go run . artisan tenant:orphans --drop   # drop them after confirmation (with their paired t_<code> account); irreversible, back up first
```

Only databases with the `TENANCY_DATABASE_PREFIX` / `TENANCY_SCHEMA_PREFIX` prefix are considered; the platform database and reserved schemas are never dropped. `t_` accounts without an orphaned database are listed, never dropped (a DBA may have created them).

### 建库与密码

- `host` 空：在平台实例 `CREATE`
- `host` 有值：连目标主机系统库再 `CREATE`
- `--skip-create` / `skip_create`：库已存在，只登记
- `tenants.password`：仅 `enc:v1:` + `APP_KEY` 密文；无明文兼容
- HTTP `POST /api/platform/tenants`：**禁止** `migrate=true`（返回 `tenant_migrate_via_cli`）

## Log types (do not mix)

| Type | Storage | Answers | Where to view |
|------|---------|---------|---------------|
| Operation log | Tenant DB / platform `platform_operation_logs` | Who changed what (audit) | Tenant admin / Platform operation logs |
| Tenant ops log | Platform `tenant_op_logs` | Did migrate/seed/backup succeed | Platform tenant ops logs |
| System log | Each connection's `system_logs` | Panics, DB failures, infra errors | Tenant admin; Platform system logs (read-only, optional `tenant_code`) |

Platform process errors (no tenant context) write to the landlord `system_logs` via `SystemLogOrmQuery`. Tenant-bound errors stay in the tenant DB -- no dual-write to the platform DB.

## 平台 API

| Method | Path | 说明 |
|--------|------|------|
| POST | `/api/platform/login` | 登录 |
| GET | `/api/platform/info` | 当前管理员 |
| POST | `/api/platform/logout` | 登出 |
| GET/POST | `/api/platform/tenants` | List / create (no sync migrate) |
| GET/PUT | `/api/platform/tenants/{id}` | Detail / update connection |
| PUT | `/api/platform/tenants/{id}/status` | Enable/disable |
| POST | `/api/platform/tenants/{id}/ping` | Ping tenant DB |
| POST | `/api/platform/tenants/{id}/migrate` | Async migrate (`with_seed` optional) |
| POST | `/api/platform/tenants/{id}/seed` | Async seed |
| POST | `/api/platform/tenants/{id}/backup` | Async backup |
| GET | `/api/platform/tenants/{id}/backups` | List backup files under `storage/backups/tenants/{code}/` |
| GET | `/api/platform/tenants/{id}/backups/download?name=` | Download a `.sql` backup |
| POST | `/api/platform/tenants/{id}/restore` | Async restore (`backup_name`) |
| POST | `/api/platform/tenants/{id}/backups/prune` | Keep newest N backups (`keep`) |
| DELETE | `/api/platform/tenants/{id}` | Soft-delete (`confirm_code`; optional `drop_database`; async `purge_objects` / `purge_backups`; legacy `purge_files` = both) |
| POST | /api/platform/tenants/{id}/undelete | Restore soft-deleted metadata |
| POST | /api/platform/tenants/{id}/purge | Retry async object/backup purge |
| DELETE | /api/platform/tenants/{id}/force | Hard-delete recycle-bin row (confirm_code; frees code) |
| GET | `/api/platform/tenants/{id}/overview` | DB snapshot stats |
| GET | `/api/platform/tenant-op-logs` | Platform-wide ops execution logs (filter code/op/status/batch_id/operator) |
| GET | `/api/platform/login-logs` | Platform login logs (landlord) |
| GET/POST | `/api/platform/admins` | Platform admins list / create (owner write) |
| GET/PUT/DELETE | `/api/platform/admins/{id}` | Platform admin detail / update / soft-delete |
| POST | `/api/platform/admins/{id}/reset-password` | Reset platform admin password |
| GET | `/api/platform/login-logs/{id}` | Platform login log detail |
| GET | `/api/platform/operation-logs` | Platform operation logs (landlord) |
| GET | `/api/platform/operation-logs/{id}` | Platform operation log detail |
| GET | `/api/platform/system-logs` | Read-only system logs (platform DB; `tenant_code` switches to that tenant DB) |
| GET | `/api/platform/system-logs/module-options` | Module filter options for current scope |
| GET | `/api/platform/system-logs/{id}` | System log detail (same scope as list) |
| GET | `/api/platform/tenants/{id}/system-log-summary` | Last 24h error/warning counts + recent errors |
| GET | `/api/platform/tenants/{id}/op-logs` | Ops timeline |
| GET | `/api/platform/tenants/{id}/login-links` | How to open tenant admin |
| GET | `/api/platform/tenants/settings` | Console settings (`backup_keep`, queue) |
| GET | `/api/platform/tenants/queue-status` | `long-running` queue depth |
| GET | `/api/platform/tenants/export` | Filtered CSV export |
| POST | `/api/platform/tenants/ops-batch` | Batch migrate/seed/backup |

Platform console supports per-tenant ping/migrate/seed/backup/restore/delete, batch seed/backup, and CSV export. Heavy `migrate-all` remains CLI-friendly.

公开支付回调（非 platform）：

| Method | Path | 说明 |
|--------|------|------|
| POST | `/api/payment/notify/{type}/{tenant}` | wechat\|alipay；路径绑定租户 |
| POST | `/api/payment/notify/{type}` | tenancy 关闭时可用；开启时 `tenant_required` |

## Code conventions

| API | Use |
|-----|------|
| `tenancy.Enabled()` | Database-per-tenant on? |
| `OrmQuery(ctx)` / `OrmTransaction(ctx, fn)` | Tenant business data; a nil or unbound ctx fails with `ErrNoTenantContext` when tenancy is on (migrate / seed goroutine bindings excepted) |
| `PlatformOrmQuery(ctx)` / `PlatformQuery()` | Platform data (with / without a ctx); never `PlatformOrmQuery(nil)` |
| `tenancyctx.Go(ctx, name, fn)` | Start background work: keeps the tenant, drops request cancellation, recovers panics |
| `tenancy.CacheKey` / `StoragePrefix` | Cache key / object storage prefix |
| `models.PlatformModel` | Marks platform-table models; tenant connections refuse their tables (`tenantdb.ErrPlatformTable`) |
| `TenantConnectionService.TenantContext(t)` | Platform code reading one tenant's data: a ctx bound to that tenant |
| `RunTenantScope` / `--tenant` | CLI per tenant |
| `SchemaHasTable` | Does the ctx tenant have this table? |
| `WithSchemaContext(ctx, fn)` | Run `facades.Orm()` / `facades.Schema()` as the ctx tenant on the current goroutine (only code that takes no ctx: migration-style code, seeders) |

### 硬性规则

1. Business code uses `OrmQuery(ctx)`; platform routes do **not** use the `Tenant` middleware; platform data uses `PlatformOrmQuery(ctx)` / `PlatformQuery()` (pinned platform connection). See the [developer guide](/en/advanced/tenancy-dev).
2. 隔离是**切库/切 Schema**，不是行级 `tenant_id` GlobalScope。
3. 缓存/上传（含 `chunks/`、导出、导入与附件临时目录）走租户前缀。
4. Search indexes are `{code}_orders` when bound (the full code, never truncated); unbound fails closed and never falls back to the shared `orders`. Before a tenant is deleted for good (platform force-delete / `tenant:cleanup-deleted` / delete after purge) all its search indexes are dropped; if that fails the tenant row stays and the code is not freed (a new tenant reusing the code must not read the old tenant's documents). Creating a tenant also clears indexes left under its code.
5. 导出/搜索队列缺 `tenant_id` 时 fail-closed，禁止写到平台库。
6. HTTP 手动跑定时任务自动带当前 `--tenant`，禁止扫全租户；cron 可遍历启用租户。
7. 队列表 `jobs` / `failed_jobs` 读平台连接（`QUEUE_DATABASE_CONNECTION`）。
8. Browser CORS: `CORS_ALLOWED_HEADERS` must include `X-Tenant-ID`; subdomains use `TENANCY_BASE_DOMAIN` wildcards; for vanity hosts prefer same-Host or demo empty/`*` (see CORS above); on split API set `ADMIN_ORIGIN_GUARD=true` (see Origin guard) — do not hand-edit the CORS allowlist per merchant.
9. 订单搜索用 `search:*` / `SyncOrderSearch`（`SEARCH_*`），勿再接旧 ES outbox 链路。
10. IP 黑名单：进程内短 TTL（约 30s）缓存启用名单；CRUD 后立即失效。查库失败时在约 5 分钟内回退最近成功缓存，超时仍 **fail-closed**（503）。
11. Only `provision_status=ready` tenants may bind traffic; HTTP create forbids sync migrate — use platform UI async migrate or CLI `tenant:migrate`.
12. 远程租户库禁止空账号回落平台 root；公网默认 `TENANCY_ALLOW_PLATFORM_DB_CREDENTIALS=false`。
13. 平台表迁移不得落在租户库（`SkipOnTenantConnection`）；migrate 失败须可在平台侧看到 `last_migrate_error`。
14. PG schema 隔离的 backup/restore 必须限定 schema；登录对 `tenant_not_ready` 返回 403（非 500）。
15. 公网优先 subdomain；支付回调必须带 `{type}/{tenant_code}` 路径。
16. Vanity domains: edge Host rewrite (see above); do not add per-domain app env, CORS allowlist entries, or separate deploys.
17. CI runs `go run ./scripts/tenantlint`, which checks four rules per call site: `orm` (no `facades.Orm().Query/Transaction/WithContext/Connection` under `app/`), `nilctx` (no `OrmQuery(nil)` / `PlatformOrmQuery(nil)` ...), `cache` (a function using `facades.Cache()` goes through `tenancy.CacheKey`, possibly via a key helper of the same package) and `goctx` (a `go` statement must not capture a request ctx; use `tenancyctx.Go`). An intended exception carries `//tenantlint:allow <rule>: <reason>` at the call site or in the function's doc comment. Data statements in migrations (`database/migrations`) use `migrationQuery()`: it runs inside the migration's transaction, `facades.Orm()` does not (on PostgreSQL it cannot see a table the migration just created).
18. Platform tables (`models.PlatformModel`) must match the tables landlord-only migrations create (`TestPlatformTablesMatchLandlordOnlyMigrations`); tenant connections refuse ORM statements on them.
19. `TestEveryAdminRouteIsTenantIsolated` marks tenant A's data and calls every GET route as tenant B, asserting the marker never shows; new routes are included automatically. `TestTenantResolutionSecurity` covers host/header conflicts, the public apex ignoring client-supplied tenants, platform/tenant tokens not crossing, disable/maintenance taking effect at once, and a reused code not honouring the old tokens.
20. **A reused code inherits nothing**: a permanent delete removes the database, storage, backups and indexes before freeing the code; provisioning refuses an existing database / schema of the same name (`tenant_storage_exists`) instead of adopting it; an empty database prepared by a DBA is taken over with `skip_create`. A failed create removes its row for good (and what it created), so the code is reusable at once.
21. **Tenant resolution status codes**: no tenant given 400; no such tenant 404; disabled or not ready 403; maintenance 503; tenant or platform database unreachable **503 + `Retry-After: 5`** (`tenant_connection_failed` / `tenant_lookup_failed`), so a platform outage is never reported as 404; misconfiguration such as missing credentials 500 with its own code. Probes and CDNs can tell "no such tenant" from "database down".

## Platform ops: domain board / onboard / health

- Tenant list: domain_status + domain_host search; health_status.
- Onboard: POST /api/platform/tenants/onboard.
- Health cron: 	enant:health-inspect hourly; webhook + optional TENANT_HEALTH_ALERT_MAIL.
- Manual: POST /api/platform/tenants/health-inspect.
