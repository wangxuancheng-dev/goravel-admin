# 多租户：一户一库 / Schema（MySQL + PostgreSQL）

默认 **`TENANCY_DRIVER=off`**：整站单库，与改造前用法相同，无需 Header。

设为 **`database`** 后：每个租户独立 database（MySQL）或 database/schema（PostgreSQL）；**租户认证与业务都在该租户库**（先解析租户，再 JWT）。

平台控制台（Landlord）使用**另一套账号与 API**，只操作平台库，不进入租户库。

## 架构

```text
TENANCY_DRIVER=off     → 默认 DB_* 单库（无平台/租户拆分）

TENANCY_DRIVER=database
  ├── 平台库 DB_*
  │     ├── tenants（开户 / Host·Port·凭据 / 启停）
  │     ├── platform_admins（平台控制台账号）
  │     └── personal_access_tokens（平台 token，tokenable_type=platform_admin）
  └── 租户库（按 tenants 连接）
        ├── admins / RBAC / 业务表
        └── personal_access_tokens（租户管理员 token）
```

### 两套控制台

| | 租户后台 | 平台控制台 |
|--|----------|------------|
| 入口 | `/login` | `/platform/login` |
| API 前缀 | `/api/admin` | `/api/platform` |
| Token | `token`（租户库） | `platform_token`（平台库） |
| 中间件 | `Tenant` → `Jwt` | `RequireTenancy` → `PlatformJwt`（**不切租户库**） |
| 职责 | 商户业务 / RBAC | 开户、远程连接、启停租户 |

**禁止**在租户后台再挂「租户管理」菜单；旧种子会在 `MenuSeeder` 中把 `slug=tenant` 下线。

### 请求解析租户（仅 `/api/admin`）

1. `TENANCY_RESOLVER=header`（默认）：Header `X-Tenant-ID` / Query `tenant_id` / `tenant_code` / 登录体 `tenant_code`
2. `TENANCY_RESOLVER=subdomain`：优先 Host 一级子域（如 `acme.example.com` → `acme`），再回落 Header/Query；保留标签见 `TENANCY_SUBDOMAIN_RESERVED`

## 配置

```ini
TENANCY_DRIVER=off                 # off | database
TENANCY_RESOLVER=header            # header | subdomain
TENANCY_HEADER=X-Tenant-ID
TENANCY_SUBDOMAIN_RESERVED=www,api,admin,platform,static,assets
TENANCY_DATABASE_PREFIX=tenant_
TENANCY_SCHEMA_PREFIX=tenant_
```

前端（双端）：

```ini
VITE_TENANCY_ENABLED=true
# 或 VITE_TENANCY_DRIVER=database
VITE_TENANCY_HEADER=X-Tenant-ID
# 可选：平台 API 前缀（默认 /api/platform）
# VITE_PLATFORM_API_PREFIX=/api/platform
```

## 首次启用步骤

```bash
# 1) 平台库迁移（含 tenants / platform_admins / tokens 等）
go run . artisan migrate

# 2) 创建平台管理员
go run . artisan platform:admin admin 'your-password' --name=平台管理员

# 3) 开户（本地同机或远程凭据）
go run . artisan tenant:create acme "Acme Corp" --driver=mysql --isolation=database --migrate
go run . artisan tenant:create remote "Remote" \
  --host=10.0.0.8 --port=3306 --username=tenant_u --password=secret \
  --database=tenant_remote --migrate

# 远程库已由 DBA 建好时：
go run . artisan tenant:create remote2 "Remote2" \
  --host=10.0.0.8 --username=tenant_u --password=secret \
  --database=tenant_remote2 --skip-create --migrate

# 4) 前端打开 /platform/login 用平台账号管理租户；
#    商户打开 /login 填租户码登录该租户库 admins
```

已有租户库需去掉旧「租户管理」菜单时：

```bash
go run . artisan tenant:seed acme --class=MenuSeeder --class=PermissionSeeder
# 或
go run . artisan tenant:seed-all --class=MenuSeeder
```

## 命令一览

```bash
go run . artisan migrate
go run . artisan platform:admin {username} {password} [--name=...]

go run . artisan tenant:create {code} {name} \
  [--driver=mysql|postgres] [--isolation=database|schema] \
  [--host=] [--port=] [--username=] [--password=] \
  [--database=] [--schema=] [--skip-create] [--migrate]

go run . artisan tenant:migrate {id|code}
go run . artisan tenant:migrate-all
go run . artisan tenant:seed {id|code} [--class=...] [--seeder=...]
go run . artisan tenant:seed-all [--class=...]
go run . artisan tenant:list
go run . artisan tenant:enable {id|code}
go run . artisan tenant:disable {id|code}

# 依赖本机 mysqldump/mysql 或 pg_dump/psql
go run . artisan tenant:backup {id|code}
go run . artisan tenant:restore {id|code} {path/to.sql}
```

`--migrate` 会在 migrate 后自动完整 `tenant:seed`。只补某一类数据用 `--class` / `--seeder`。

远程库：`host/port/username/password` 写在 `tenants` 表；空字段回落平台 `DB_*`。

- **建库位置**：`host` 为空 → 在平台库实例上 `CREATE`；`host` 有值 → 用租户凭据连接**该主机**系统库（`mysql` / `postgres`）再 `CREATE`。
- **库已存在**：API/`tenant:create` 传 `skip_create` / `--skip-create`，跳过 CREATE，只登记并 migrate。
- **密码落库**：`tenants.password` 使用 `APP_KEY`（`Crypt.EncryptString`）加密，前缀 `enc:v1:`；旧明文仍可读。API 只返回 `has_password`。

平台 API 返回 `has_password`，不回传明文密码。

## 平台 API

| Method | Path | 说明 |
|--------|------|------|
| POST | `/api/platform/login` | 平台登录 |
| GET | `/api/platform/info` | 当前平台管理员 |
| POST | `/api/platform/logout` | 登出 |
| GET | `/api/platform/tenants` | 租户列表 |
| GET | `/api/platform/tenants/{id}` | 详情 |
| POST | `/api/platform/tenants` | 开户（可带远程凭据 + migrate） |
| PUT | `/api/platform/tenants/{id}` | 更新名称/连接元数据 |
| PUT | `/api/platform/tenants/{id}/status` | 启停 |

## 代码约定（必读）

| API | 用途 |
|-----|------|
| `tenancy.Enabled()` | 是否一户一库；**唯一开关判断** |
| `tenancy.Resolver()` / `HTTPHint` / `SubdomainHint` | 租户解析 |
| `tenancy.CacheKey(ctx, key)` | 缓存/锁键租户前缀 |
| `tenancy.StoragePrefix(ctx)` | 对象存储路径前缀 `tenants/{code}/` |
| `OrmQuery(ctx)` | **租户业务与认证默认入口** |
| `PlatformOrmQuery(ctx)` | **仅**平台元数据（tenants / platform_admins / 平台 tokens） |
| `NewPlatformTokenService` | 平台 token（平台库） |
| `WithTenantConnection` | migrate / seed 共用串行切库 |

### 硬性规则

1. 租户业务优先 `appfacades.OrmQuery(ctx)`，不要直接 `facades.Orm().Query()`。
2. 平台元数据只用 `PlatformOrmQuery`；平台控制台路由**不要**挂 `Tenant` 中间件。
3. 缓存键、分布式锁经 `tenancy.CacheKey`。
4. 上传路径经 `tenancy.StoragePrefix`。
5. 不要再加行级 `tenant_id` GlobalScope。

导出任务 `ExportArgs` 携带 `tenant_id`，worker 先 `JobContext` 再拿执行锁与写库。

## 前端 / 调用方

开启 tenancy 后：

1. **租户登录** `/login`：显示租户码，body `tenant_code`，Header `X-Tenant-ID`；公开图片带 `?tenant_code=`
2. **平台登录** `/platform/login`：独立 `platform_token`，请求 `/api/platform/*`，**不带**租户 Header
3. URL 预填：`/login?tenant_code=acme`
4. 子域解析开启时，可用 `acme.yourdomain.com` 省略 Header（仍建议登录页写入本地码供静态资源）

## 已废弃

- 共享表 + `WHERE tenant_id` / GlobalScope 行级方案
- 「认证在平台库、业务在租户库」的折中中间态
- 租户后台内嵌「租户管理」CRUD（已迁至 `/api/platform` + `/platform/*`）
