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

## 配置

```ini
TENANCY_DRIVER=database
TENANCY_RESOLVER=header            # header | subdomain
TENANCY_HEADER=X-Tenant-ID
TENANCY_SUBDOMAIN_RESERVED=www,api,admin,platform,static,assets
TENANCY_DATABASE_PREFIX=tenant_
TENANCY_SCHEMA_PREFIX=tenant_

PLATFORM_ADMIN_USERNAME=admin
PLATFORM_ADMIN_PASSWORD=secret
PLATFORM_ADMIN_NAME=平台管理员
```

前端：`VITE_TENANCY_ENABLED=true`（或 `VITE_TENANCY_DRIVER=database`）。

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
go run . artisan tenant:backup {id|code}
go run . artisan tenant:restore {id|code} {sql路径}
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

## 代码约定

| API | 用途 |
|-----|------|
| `tenancy.Enabled()` | 是否一户一库 |
| `OrmQuery(ctx)` | 租户业务默认入口 |
| `PlatformOrmQuery(ctx)` | 平台元数据 / 平台 token |
| `NewPlatformTokenService` | 平台 token |

硬性规则：业务用 `OrmQuery`；平台路由不挂 `Tenant` 中间件；缓存/上传走 `tenancy.CacheKey` / `StoragePrefix`。
