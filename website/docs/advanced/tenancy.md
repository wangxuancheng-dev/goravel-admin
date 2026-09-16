# 多租户：一户一库 / Schema（MySQL + PostgreSQL）

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
| 开户 migrate | — | 平台 UI 异步入队 / CLI；HTTP 开户仍禁止同步 migrate |
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

## Docker 本地开启

默认 `docker compose` / [快速开始](/guide/getting-started) 为**单库**（`TENANCY_DRIVER=off`）。本地要试用一户一库时：

1. 先按快速开始把栈跑起来（`.env` 来自 `.env.docker.example`）。
2. 在根目录 `.env` 中增加或改成：

```ini
TENANCY_DRIVER=database
TENANCY_RESOLVER=header
TENANCY_HEADER=X-Tenant-ID
# 同机 MySQL 容器可共用平台库账号建 tenant_*（仅本地）
TENANCY_ALLOW_PLATFORM_DB_CREDENTIALS=true

PLATFORM_ADMIN_USERNAME=admin
PLATFORM_ADMIN_PASSWORD=secret
PLATFORM_ADMIN_NAME=平台管理员
```

3. 重启 API 容器使配置生效：

```bash
docker compose up -d app
# 或
docker compose restart app
```

4. 平台首启 + 开示例租户（容器内二进制为 `/www/main`）：

```bash
docker compose exec app /www/main artisan platform:install
docker compose exec app /www/main artisan tenant:create acme "Acme" --migrate
```

5. 前端本地开发时开启租户提示（`html/.env` 或 `html-react/.env`）：

```ini
VITE_TENANCY_ENABLED=true
VITE_TENANCY_DRIVER=database
VITE_TENANCY_HEADER=X-Tenant-ID
```

6. 访问：

| 入口 | 说明 |
|------|------|
| `/platform/login` | 平台控制台（`PLATFORM_ADMIN_*`） |
| `/login` + Header `X-Tenant-ID: acme`，或 `/login?tenant_code=acme` | 租户后台（租户库管理员，以 seed 为准） |

开户后可在平台租户列表点 **迁移**（异步，可选 seed），无需再跑 CLI；`QUEUE_CONNECTION=sync` 时任务在请求内同步执行，生产请用 Redis + `long-running` worker。

注意：

- 公网请改用 `TENANCY_RESOLVER=subdomain`，并保持 `TENANCY_ALLOW_PLATFORM_DB_CREDENTIALS=false`。
- Compose 默认仍是演示单库。

## 生产要点

1. **平台连接钉死**：`PlatformOrmQuery` 使用 `tenancy.platform_connection`，不跟随 migrate 时临时翻转的 `database.default`。
2. **生产 Artisan**：`APP_ENV=production` 白名单含 `tenant:*` / `platform:*`（开户/迁移可用）。
3. **连接回收**：`Forget` 会 `Close` + `Fresh` 动态连接池。
4. **开户状态**：HTTP/CLI 创建后为 `pending`；平台 UI 异步迁移或 CLI `tenant:migrate` → `migrating` → `ready`/`failed`；未 ready 禁止业务绑定。UI 入队后若 worker 未消费，约 30 分钟后允许重试（防永久卡住）。
5. **账号隔离**：远程库必须独立凭据；同机共用平台账号仅当 `TENANCY_ALLOW_PLATFORM_DB_CREDENTIALS=true`（**公网默认 false**）。
6. **异步运维**：单户 migrate / seed / backup / restore 走 `tenant_ops`（`long-running`）；生产需 Redis 队列 + long-running worker。`migrate-all` / `backup-all` 仍可用 CLI。

## 公网部署（推荐）

1. **`TENANCY_RESOLVER=subdomain`**：租户以 `acme.example.com` 访问；apex/`www`/`platform` 等保留域**不接受** Header/Query 冒充（除非显式 `TENANCY_ALLOW_HEADER_FALLBACK=true`）。
2. 子域与 Header/body 冲突 → `tenant_hint_conflict`（400）。
3. **支付回调**：`POST /api/payment/notify/{type}/{tenant_code}`（渠道不会带租户 Header）；tenancy 开启时须带 `{tenant_code}`。
4. **连接池**：每租户 `TENANCY_POOL_MAX_*`（默认 idle 2 / open 20）；户多时按下方 [规模与推荐配置](#规模与推荐配置) 下调，避免打满 MySQL。
5. 平台控制台走 `platform.` 或独立域名；勿与租户子域混用。



## 商户独立域名（tenant_domains）

子域 `{code}.${TENANCY_BASE_DOMAIN}` 默认可用。独立域名在**平台控制台**绑定，无需按户改 Nginx / 重启 API。

| 场景 | 做法 |
|------|------|
| 大多数商户 | 泛解析 `*.example.com` + 子域 |
| 独立域（无 CDN） | `ssl_mode=edge`：CNAME 到 `TENANCY_DOMAIN_TARGET`，边缘 on-demand 出证 |
| 自有 CDN+SSL | `ssl_mode=customer_cdn`：CDN 回源，**回源 Host 保持客户域名** |

解析优先级：`active` 自定义 Host → 子域 → Header/Query。

公网启用独立域名时请设置 `TENANCY_BASE_DOMAIN`：子域解析仅认 `{code}.该主域`，避免把 `crm.客户域.com` 误当成租户短码。C 端 `/api/user`、`/api/public/*` 与后台共用 Host 绑定。

```ini
TENANCY_BASE_DOMAIN=example.com
TENANCY_DOMAIN_TARGET=tenants.example.com
TENANCY_DOMAIN_VERIFY_PREFIX=_goravel-tenant
TENANCY_DOMAIN_CACHE_TTL=60
```

平台 API：`GET/POST /api/platform/tenants/{id}/domains`，`POST .../verify`，`PUT .../primary|disable`，`DELETE`；边缘询问 `GET /api/platform/public/tls-allow?host=`（仅 edge+active 返回 200）。

旧方案（Nginx 改写 Host 为 `{code}.主域`）仍可用；新产品请用 `tenant_domains`。
## 规模与推荐配置

以下为**经验起点**，需按监控回调，不是硬性配额。户多时首要瓶颈通常是**租户库连接池**（不是 Redis）。Redis 全站共用（缓存键经 `tenancy.CacheKey` 加 `t{id}:` 前缀），一般升规格即可；队列与缓存吵邻居时再考虑拆实例或分 DB。

**容量公式（租户库）：**

`进程内已注册的租户池数 × TENANCY_POOL_MAX_OPEN_CONNS × API 实例数` ≪ 数据库 `max_connections`（预留平台库、备份、运维余量）。

「已注册池数」≈ 近期有流量、尚未被 Forget 的租户，**不是** `tenants` 表总行数。同机多库时所有 `tenant_*` 仍计入同一 MySQL/PG 实例的连接上限。

| 活跃商户量级（经验） | 租户池建议 | 队列 / 进程 | Redis |
|----------------------|------------|-------------|-------|
| &lt; 50 | 默认 `IDLE=2` / `OPEN=20` 可留；流量低可先降到 `OPEN=10` | 可同机 Worker；`QUEUE_CONNECTION=redis` | 单机或小规格云 Redis |
| 50–200 | `IDLE=1–2`，`OPEN=3–5`；缩短 idle/lifetime（如 120 / 600） | API 与 Worker **分角色**（见 [生产清单](/deploy/production) §4.1）；`QUEUE_LONG_RUNNING_CONCURRENT` 保持较小，靠加 Worker 机水平扩 | 托管 Redis，盯 `used_memory` 与队列 backlog |
| 200+ | `OPEN` 进一步收紧或扩库 `max_connections`；避免默认 20 原样上生产 | 独立 Worker 消费 `default` + `long-running`；配置 `QUEUE_ALERT_BACKLOG_THRESHOLD` | 更大规格 / Cluster；导出导入高峰注意吵邻居 |

示例（约 100 活跃户、2 个 API 实例时的起点）：

```ini
TENANCY_POOL_MAX_IDLE_CONNS=1
TENANCY_POOL_MAX_OPEN_CONNS=5
TENANCY_POOL_CONN_MAX_IDLETIME=120
TENANCY_POOL_CONN_MAX_LIFETIME=600

CACHE_STORE=redis
QUEUE_CONNECTION=redis
QUEUE_CONCURRENT=2
QUEUE_LONG_RUNNING_CONCURRENT=1
# API 机：APP_DISABLED_RUNNERS=queue-*
```

**建议监控：** MySQL/PG `Threads_connected`（或等价指标）、队列 pending / `queue:alert-backlog`、Redis 内存与连接数。接近上限时先下调 `TENANCY_POOL_*` 或扩容，而不是盲目加 API 副本（副本会放大连接占用）。


## 对象存储与配额

- **共用 disk**：S3/OSS/本地等仍是全站一份 `FILESYSTEM_DISK`；路径用 `tenants/{code}/` 前缀隔离。租户后台**不可**改 `file_disk`（保存会被拒绝），仅可改导出格式。
- **删除租户**：平台元数据**软删**入回收站；drop_database 仅 DROP 库/Schema；**此时不清理对象存储**。
- **回收站 / 永久删除**：GET /tenants?trashed=only；POST .../undelete 恢复；POST .../purge 可单独重试清理；DELETE .../force 默认异步清理对象+备份后再硬删并释放 code。保留 TENANCY_DELETED_RETENTION_DAYS；定时 	enant:cleanup-deleted。
- **存储限额** `storage_limit_bytes`（0=不限）：按租户库 `attachments.size` 汇总，上传前校验。

## 运维增强

1. **Landlord 迁移跳过**：平台表迁移（`tenants` / `platform_admins` / `jobs` / provision/migrate meta）在 `tenant_*` 连接上 `SkipOnTenantConnection` 空跑，避免污染租户库。
2. **Migrate / 运维可见性**：`last_migrate_error` / `migrated_at`；平台 UI 另有 `last_op` / `last_op_status` / `last_op_message` / `last_backup_path`。失败写 `provision_status=failed`（seed 失败不降级已 ready）。
3. **连接探测 / 异步操作**：`POST .../ping`；`.../migrate|seed|backup` 入队。
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
| GET/POST | `/api/platform/tenants` | 列表 / 开户（无同步 migrate） |
| GET/PUT | `/api/platform/tenants/{id}` | 详情 / 更新连接 |
| PUT | `/api/platform/tenants/{id}/status` | 启停 |
| POST | `/api/platform/tenants/{id}/ping` | 探测租户库连通性 |
| POST | `/api/platform/tenants/{id}/migrate` | 异步 migrate（body 可选 `with_seed`） |
| POST | `/api/platform/tenants/{id}/seed` | 异步 seed |
| POST | `/api/platform/tenants/{id}/backup` | 异步备份 |
| GET | `/api/platform/tenants/{id}/backups` | 备份文件列表（`storage/backups/tenants/{code}/`） |
| GET | `/api/platform/tenants/{id}/backups/download?name=` | 下载指定 `.sql` 备份 |
| POST | `/api/platform/tenants/{id}/restore` | 异步从备份恢复（body `backup_name`） |
| POST | `/api/platform/tenants/{id}/backups/prune` | 保留最新 N 份备份（body `keep`） |
| DELETE | `/api/platform/tenants/{id}` | 软删元数据（`confirm_code`；可选 `drop_database`；`purge_objects` / `purge_backups` **异步**清文件，兼容 `purge_files`=两者） |
| POST | /api/platform/tenants/{id}/undelete | 从回收站恢复元数据 |
| POST | /api/platform/tenants/{id}/purge | 重试异步清理对象/备份 |
| DELETE | /api/platform/tenants/{id}/force | 永久删除回收站记录（confirm_code；释放 code） |
| GET | `/api/platform/tenants/{id}/overview` | 库概览（表数、体积、管理员数等） |
| GET | `/api/platform/tenant-op-logs` | 全平台运维执行记录（筛选 code/op/status/batch_id/operator） |
| GET | `/api/platform/tenants/{id}/op-logs` | 运维时间线 |
| GET | `/api/platform/tenants/{id}/login-links` | 租户后台登录方式 |
| GET | `/api/platform/tenants/settings` | 控制台可见配置（`backup_keep`、队列） |
| GET | `/api/platform/tenants/queue-status` | `long-running` 队列深度 |
| GET | `/api/platform/tenants/export` | 按筛选导出 CSV |
| POST | `/api/platform/tenants/ops-batch` | 批量 migrate/seed/backup |

平台控制台列表可操作单户 **Ping / 迁移 / 种子 / 备份 / 恢复 / 删除**（入队 `tenant_ops`，`long-running` 队列），并支持批量 seed/backup、CSV 导出。`migrate-all` 等仍可用 CLI。

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
11. 仅 `provision_status=ready` 的租户可绑定业务；HTTP 开户禁止同步 migrate，可走平台 UI 异步迁移或 CLI `tenant:migrate`。
12. 远程租户库禁止空账号回落平台 root；公网默认 `TENANCY_ALLOW_PLATFORM_DB_CREDENTIALS=false`。
13. 平台表迁移不得落在租户库（`SkipOnTenantConnection`）；migrate 失败须可在平台侧看到 `last_migrate_error`。
14. PG schema 隔离的 backup/restore 必须限定 schema；登录对 `tenant_not_ready` 返回 403（非 500）。
15. 公网优先 subdomain；支付回调必须带 `{type}/{tenant_code}` 路径。
16. 商户独立域名用边缘改写 Host（见上文）；勿为每个独立域改应用 env 或独立部署。
17. 业务目录（`app/services` / `http` / `jobs` / `console`）禁止 `facades.Orm().Query()`；CI 跑 `bash scripts/check-tenant-orm.sh`（租户运维白名单除外）。
