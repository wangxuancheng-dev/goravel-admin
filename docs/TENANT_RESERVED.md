# 多租户：一户一库 / Schema（MySQL + PostgreSQL）

默认 **`TENANCY_DRIVER=off`**：整站单库。

设为 **`database`** 后：每租户独立 database（MySQL）或 database/schema（PostgreSQL）；**租户认证与业务都在该租户库**。平台控制台使用独立账号与 `/api/platform`，不切租户库。

> 按新项目开发：不做旧版菜单/明文密码兼容。

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
| 开户 migrate | — | **仅 CLI**（HTTP 只登记/建库） |
| 开户状态 | — | `provision_status`: `pending` → `ready`（`tenant:migrate` 成功后）；未 ready 禁止业务绑定 |

## 配置

```ini
TENANCY_DRIVER=database
TENANCY_RESOLVER=header            # header | subdomain
TENANCY_HEADER=X-Tenant-ID
TENANCY_SUBDOMAIN_RESERVED=www,api,admin,platform,static,assets
TENANCY_DATABASE_PREFIX=tenant_
TENANCY_SCHEMA_PREFIX=tenant_
TENANCY_PLATFORM_CONNECTION=       # 可选；钉死平台连接名，默认取 database.default
TENANCY_ALLOW_PLATFORM_DB_CREDENTIALS=true  # 同机空账号回落平台 DB_*；生产建议 false
                                          # 远程 host 始终要求独立 username/password
TENANCY_POSTGRES_SSLMODE=                 # 空则回落 DB_SSLMODE / disable
TENANCY_BACKUP_KEEP=10                    # tenant:backup 保留份数；0=不清理

PLATFORM_ADMIN_USERNAME=admin
PLATFORM_ADMIN_PASSWORD=secret
PLATFORM_ADMIN_NAME=平台管理员

# header 解析租户时，浏览器跨域需放行（默认已含）：
# CORS_ALLOWED_HEADERS=...,X-Tenant-ID
```
前端：`VITE_TENANCY_ENABLED=true`（或 `VITE_TENANCY_DRIVER=database`）。

## 生产要点（P0）

1. **平台连接钉死**：`PlatformOrmQuery` 使用 `tenancy.platform_connection`，不跟随 migrate 时临时翻转的 `database.default`。
2. **生产 Artisan**：`APP_ENV=production` 白名单含 `tenant:*` / `platform:*`（开户/迁移可用）。
3. **连接回收**：`Forget` 会 `Close` + `Fresh` 动态连接池。
4. **开户状态**：HTTP/CLI 创建后为 `pending`；`tenant:migrate` 成功 → `ready`；未 ready 的租户不可绑定业务请求，也不可启用以绕过。
5. **账号隔离**：远程库必须独立凭据；同机共用平台账号仅当 `TENANCY_ALLOW_PLATFORM_DB_CREDENTIALS=true`。

## 运维增强（P1）

1. **Landlord 迁移跳过**：平台表迁移（`tenants` / `platform_admins` / `jobs` / provision/migrate meta）在 `tenant_*` 连接上 `SkipOnTenantConnection` 空跑，避免污染租户库。
2. **Migrate 可见性**：`last_migrate_error` / `migrated_at`；失败写 `provision_status=failed`。
3. **连接探测**：`POST /api/platform/tenants/{id}/ping`。
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
go run . artisan platform:admin {username} {password} [--name=]

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
| GET/POST | `/api/platform/tenants` | 列表 / 开户（无 migrate） |
| GET/PUT | `/api/platform/tenants/{id}` | 详情 / 更新连接 |
| PUT | `/api/platform/tenants/{id}/status` | 启停 |
| POST | `/api/platform/tenants/{id}/ping` | 探测租户库连通性 |

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
11. 仅 `provision_status=ready` 的租户可绑定业务；HTTP 开户禁止 migrate，须 CLI `tenant:migrate`。
12. 远程租户库禁止空账号回落平台 root；生产建议 `TENANCY_ALLOW_PLATFORM_DB_CREDENTIALS=false`。
13. 平台表迁移不得落在租户库（`SkipOnTenantConnection`）；migrate 失败须可在平台侧看到 `last_migrate_error`。
14. PG schema 隔离的 backup/restore 必须限定 schema；登录对 `tenant_not_ready` 返回 403（非 500）。
