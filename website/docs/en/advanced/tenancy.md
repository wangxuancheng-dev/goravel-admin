# Multi-tenancy

> Chinese remains the full reference: [多租户](/advanced/tenancy). Key local Docker steps are below in English.

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
                                          # 远程 host 始终要求独立 username/password
TENANCY_POSTGRES_SSLMODE=                 # 空则回落 DB_SSLMODE / disable
TENANCY_BACKUP_KEEP=10                    # tenant:backup 保留份数；0=不清理
TENANCY_POOL_MAX_IDLE_CONNS=2
TENANCY_POOL_MAX_OPEN_CONNS=20

PLATFORM_ADMIN_USERNAME=admin
PLATFORM_ADMIN_PASSWORD=secret
PLATFORM_ADMIN_NAME=平台管理员

# header 解析租户时，浏览器跨域需放行（默认已含）：
# CORS_ALLOWED_HEADERS=...,X-Tenant-ID
```
前端：`VITE_TENANCY_ENABLED=true`（或 `VITE_TENANCY_DRIVER=database`）。

## 生产要点

1. **平台连接钉死**：`PlatformOrmQuery` 使用 `tenancy.platform_connection`，不跟随 migrate 时临时翻转的 `database.default`。
2. **生产 Artisan**：`APP_ENV=production` 白名单含 `tenant:*` / `platform:*`（开户/迁移可用）。
3. **连接回收**：`Forget` 会 `Close` + `Fresh` 动态连接池。
4. **开户状态**：HTTP/CLI 创建后为 `pending`；平台 UI 异步迁移或 CLI `tenant:migrate` → `migrating` → `ready`/`failed`；未 ready 禁止业务绑定。UI 入队后若 worker 未消费，约 30 分钟后允许重试。
5. **账号隔离**：远程库必须独立凭据；同机共用平台账号仅当 `TENANCY_ALLOW_PLATFORM_DB_CREDENTIALS=true`（**公网默认 false**）。
6. **异步运维**：单户 migrate / seed / backup / restore 走 `tenant_ops`（`long-running`）；生产需 Redis + long-running worker。`migrate-all` / `backup-all` 仍可用 CLI。

## 公网部署（推荐）

1. **`TENANCY_RESOLVER=subdomain`**：租户以 `acme.example.com` 访问；apex/`www`/`platform` 等保留域**不接受** Header/Query 冒充（除非显式 `TENANCY_ALLOW_HEADER_FALLBACK=true`）。
2. 子域与 Header/body 冲突 → `tenant_hint_conflict`（400）。
3. **支付回调**：`POST /api/payment/notify/{type}/{tenant_code}`（渠道不会带租户 Header）；tenancy 开启时须带 `{tenant_code}`。
4. **Connection pool**: per-tenant `TENANCY_POOL_MAX_*` (default idle 2 / open 20); at higher tenant counts, tighten using [Scale and recommended settings](#scale-and-recommended-settings) so MySQL is not exhausted.
5. 平台控制台走 `platform.` 或独立域名；勿与租户子域混用。

## Scale and recommended settings

These are **starting points**, not hard quotas — tune from monitoring. With many tenants the first bottleneck is usually the **per-tenant DB pool**, not Redis. Redis is shared cluster-wide (cache keys use `tenancy.CacheKey` → `t{id}:` prefix); size the instance up first. Split cache vs queue (or Redis DB indexes) only if noisy-neighbor becomes real.

**Capacity formula (tenant DBs):**

`registered tenant pools in process × TENANCY_POOL_MAX_OPEN_CONNS × API instances` ≪ DB `max_connections` (leave headroom for platform DB, backups, ops).

"Registered pools" ≈ tenants with recent traffic that have not been Forgotten — **not** the row count of `tenants`. Multiple `tenant_*` databases on one MySQL/PG host still share that host's connection limit.

| Active tenants (rule of thumb) | Tenant pool | Queue / processes | Redis |
|--------------------------------|-------------|-------------------|-------|
| &lt; 50 | Defaults `IDLE=2` / `OPEN=20` OK; low traffic can use `OPEN=10` | Worker may share the API host; `QUEUE_CONNECTION=redis` | Single or small managed Redis |
| 50–200 | `IDLE=1–2`, `OPEN=3–5`; shorter idle/lifetime (e.g. 120 / 600) | Split **API vs Worker** roles (see [Production](/en/deploy/production) §4.1); keep `QUEUE_LONG_RUNNING_CONCURRENT` low and scale Worker nodes | Managed Redis; watch `used_memory` and queue backlog |
| 200+ | Tighten `OPEN` further or raise `max_connections`; do not ship default 20 unchanged | Dedicated Workers for `default` + `long-running`; set `QUEUE_ALERT_BACKLOG_THRESHOLD` | Larger tier / Cluster; watch export/import noisy neighbors |

Example starting point (~100 active tenants, 2 API instances):

```ini
TENANCY_POOL_MAX_IDLE_CONNS=1
TENANCY_POOL_MAX_OPEN_CONNS=5
TENANCY_POOL_CONN_MAX_IDLETIME=120
TENANCY_POOL_CONN_MAX_LIFETIME=600

CACHE_STORE=redis
QUEUE_CONNECTION=redis
QUEUE_CONCURRENT=2
QUEUE_LONG_RUNNING_CONCURRENT=1
# API nodes: APP_DISABLED_RUNNERS=queue-*
```

**Watch:** MySQL/PG `Threads_connected` (or equivalent), queue pending / `queue:alert-backlog`, Redis memory and connections. Near limits, lower `TENANCY_POOL_*` or scale the DB before blindly adding API replicas (replicas multiply connection usage).


## Object storage and quotas

- **Shared disk**: one `FILESYSTEM_DISK` for the whole platform; paths are isolated with `tenants/{code}/`. Tenant admin **cannot** change `file_disk` (save rejected); export format remains editable.
- **Delete tenant**: soft-delete into recycle bin; `drop_database` drops DB/schema only; **object storage is not purged here**.
- **Recycle / force delete**: `GET /tenants?trashed=only`; `POST .../undelete`; `POST .../purge` optional retry; `DELETE .../force` defaults to async purge objects+backups then hard-delete (frees `code`). Retention `TENANCY_DELETED_RETENTION_DAYS`; schedule `tenant:cleanup-deleted`.
- **Storage limit** `storage_limit_bytes` (0=unlimited): enforced from `SUM(attachments.size)` before upload.

## 运维增强

1. **Landlord 迁移跳过**：平台表迁移（`tenants` / `platform_admins` / `jobs` / provision/migrate meta）在 `tenant_*` 连接上 `SkipOnTenantConnection` 空跑，避免污染租户库。
2. **Migrate / ops visibility**：`last_migrate_error` / `migrated_at`；platform UI also shows `last_op*` / `last_backup_path`. Seed failure does not demote `ready` → `failed`.
3. **Ping / async ops**：`POST .../ping`；`.../migrate|seed|backup` enqueue `tenant_ops`.
4. **登录限流**：`login` limiter 键含 body/query/header/`subdomain` 租户提示，避免跨租户互相锁号。
5. **日志**：带 `tenant_code` / `tenant_id` 前缀（`app/utils/logger`）。
6. **PG sslmode**：`TENANCY_POSTGRES_SSLMODE` 或 `DB_SSLMODE`。
7. **备份/恢复**：`tenant:backup [--keep=N]`、`tenant:backup-all`；PG schema 隔离备份用 `pg_dump -n`，恢复用 `PGOPTIONS=--search_path`。
8. **CLI 范围**：`RunTenantScope` 仅遍历 **active + ready**；`tenant:migrate-all` 仍可覆盖 pending（单独查询）。
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
go run . artisan tenant:migrate-all
go run . artisan tenant:seed {id|code} [--class=...]
go run . artisan tenant:seed-all
go run . artisan tenant:list
go run . artisan tenant:enable|disable {id|code}
go run . artisan tenant:backup {id|code} [--keep=N]
go run . artisan tenant:backup-all [--keep=N]
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

### 建库与密码

- `host` 空：在平台实例 `CREATE`
- `host` 有值：连目标主机系统库再 `CREATE`
- `--skip-create` / `skip_create`：库已存在，只登记
- `tenants.password`：仅 `enc:v1:` + `APP_KEY` 密文；无明文兼容
- HTTP `POST /api/platform/tenants`：**禁止** `migrate=true`（返回 `tenant_migrate_via_cli`）

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

## 代码约定

| API | 用途 |
|-----|------|
| `tenancy.Enabled()` | 是否一户一库 |
| `OrmQuery(ctx)` | 租户业务默认入口 |
| `PlatformOrmQuery(ctx)` | 平台元数据 / 平台 token |
| `tenancyctx.Detach(ctx)` | 异步 goroutine 保留租户连接 |
| `tenancy.CacheKey` / `StoragePrefix` | 缓存键 / 对象存储路径前缀 |
| `RunTenantScope` / `--tenant` | CLI 按租户执行 |
| `SchemaHasTable` / `WithSchemaContext` | 请求路径 Schema（`SchemaConnLock`） |

### 硬性规则

1. 业务用 `OrmQuery(ctx)`；平台路由**不**挂 `Tenant` 中间件；平台元数据用 `PlatformOrmQuery`（钉死平台连接）。
2. 隔离是**切库/切 Schema**，不是行级 `tenant_id` GlobalScope。
3. 缓存/上传（含 `chunks/`、导出、导入与附件临时目录）走租户前缀。
4. 搜索索引绑定后为 `{code}_orders`；未绑定 fail-closed，禁止回退共享 `orders`。
5. 导出/搜索队列缺 `tenant_id` 时 fail-closed，禁止写到平台库。
6. HTTP 手动跑定时任务自动带当前 `--tenant`，禁止扫全租户；cron 可遍历启用租户。
7. 队列表 `jobs` / `failed_jobs` 读平台连接（`QUEUE_DATABASE_CONNECTION`）。
8. 浏览器跨域 header 解析租户时，`CORS_ALLOWED_HEADERS` 须含 `X-Tenant-ID`。
9. 订单搜索用 `search:*` / `SyncOrderSearch`（`SEARCH_*`），勿再接旧 ES outbox 链路。
10. IP 黑名单：进程内短 TTL（约 30s）缓存启用名单；CRUD 后立即失效。查库失败时在约 5 分钟内回退最近成功缓存，超时仍 **fail-closed**（503）。
11. Only `provision_status=ready` tenants may bind traffic; HTTP create forbids sync migrate — use platform UI async migrate or CLI `tenant:migrate`.
12. 远程租户库禁止空账号回落平台 root；公网默认 `TENANCY_ALLOW_PLATFORM_DB_CREDENTIALS=false`。
13. 平台表迁移不得落在租户库（`SkipOnTenantConnection`）；migrate 失败须可在平台侧看到 `last_migrate_error`。
14. PG schema 隔离的 backup/restore 必须限定 schema；登录对 `tenant_not_ready` 返回 403（非 500）。
15. 公网优先 subdomain；支付回调必须带 `{type}/{tenant_code}` 路径。
