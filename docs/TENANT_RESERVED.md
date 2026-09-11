# 多租户：一户一库 / Schema（MySQL + PostgreSQL）

默认 **`TENANCY_DRIVER=off`**：整站单库，与改造前用法相同，无需 Header。

设为 **`database`** 后：每个租户独立 database（MySQL）或 database/schema（PostgreSQL）；**认证与业务都在该租户库**（先解析租户，再 JWT）。

## 架构

```text
TENANCY_DRIVER=off     → 默认 DB_* 单库
TENANCY_DRIVER=database
  ├── 平台库 DB_*：仅 tenants 元数据（开户/连库信息）
  └── 租户库：admins / RBAC / 业务表（migrate + seed）
```

请求（开启时）：`Tenant` 中间件 → `Jwt` → 业务。登录体可带 `tenant_code` / `tenant_id`，或 Header `X-Tenant-ID`。

## 配置

```ini
TENANCY_DRIVER=off          # off | database
TENANCY_HEADER=X-Tenant-ID
TENANCY_DATABASE_PREFIX=tenant_
TENANCY_SCHEMA_PREFIX=tenant_
```

## 命令

```bash
go run . artisan migrate                    # 平台库（含 tenants 表）

# MySQL 一户一库 + 迁移 + 种子（管理员/菜单/权限）
go run . artisan tenant:create acme "Acme" --driver=mysql --isolation=database --migrate

# PostgreSQL 独立库或 schema
go run . artisan tenant:create acme "Acme" --driver=postgres --isolation=database --migrate
go run . artisan tenant:create acme "Acme" --driver=postgres --isolation=schema --migrate

go run . artisan tenant:migrate acme
go run . artisan tenant:seed acme
go run . artisan tenant:seed acme --class=MenuSeeder
go run . artisan tenant:seed acme --class=MenuSeeder --class=PermissionSeeder
go run . artisan tenant:seed acme --seeder=GeneratedModulesSeeder   # 同 --class，对齐 db:seed
go run . artisan tenant:migrate-all
```

`--migrate` 会在 migrate 后自动完整 `tenant:seed`（全部 Seeder）。只补某一类数据用 `--class` / `--seeder`。

## 代码约定（必读）

| API | 用途 |
|-----|------|
| `tenancy.Enabled()` | 是否一户一库；**唯一开关判断**（`helpers`/`services` 仅别名） |
| `tenancy.CacheKey(ctx, key)` | 缓存/锁键租户前缀（登录锁定、导出锁、验证码 store key） |
| `tenancy.StoragePrefix(ctx)` | 对象存储路径前缀 `tenants/{code}/` |
| `tenancy.HTTPHint(ctx)` | Header / Query 租户标识 |
| `OrmQuery(ctx)` | **业务与认证默认入口**；ctx 已绑定时走租户库 |
| `PlatformOrmQuery(ctx)` | **仅**平台 `tenants` 与建库 DDL |
| `TenantConnectionService.BindHTTP` / `BindBackground` | HTTP / 队列切库 |
| `WithTenantConnection` | migrate / seed 共用串行切库 |

### 硬性规则

1. 业务代码优先 `appfacades.OrmQuery(ctx)`，不要直接 `facades.Orm().Query()`（会打到平台库）。
2. 平台元数据只用 `PlatformOrmQuery`；不要在业务表查询里混用。
3. 缓存键、分布式锁经 `tenancy.CacheKey`（或 `helpers.TenantCacheKey`）。
4. 上传路径经 `tenancy.StoragePrefix` / `helpers.TenantStoragePrefix`。
5. 不要再加行级 `tenant_id` GlobalScope；隔离靠独立库/schema。

导出任务 `ExportArgs` 携带 `tenant_id`，worker 先 `JobContext` 再拿执行锁与写库；`WriteData` / `Build*Query` 必须传入同一 `ctx`（勿用 `context.Background()`）。

代码生成器：`service.tpl` / `controller.tpl` 已走 `OrmQuery(ctx)`；异步导出模板为 `export_job.tpl`（与上约定一致）。前端 Header 不在生成器里注入，需在双前端 `request` 层统一处理。

## 前端 / 调用方

开启 tenancy 后（后端 `TENANCY_DRIVER=database`，前端同步设 `VITE_TENANCY_ENABLED=true` 或 `VITE_TENANCY_DRIVER=database`）：

1. 登录页显示 **租户码**，提交 body 带 `tenant_code`，并写入本地存储
2. 双前端 `request` 拦截器对后续 API（含验证码）自动加 Header `X-Tenant-ID`（可用 `VITE_TENANCY_HEADER` 改名）
3. 支持 URL 预填：`/login?tenant_code=acme` 或 `?tenant=acme`
4. 登出保留租户码，方便同一商户再次登录
5. 公开图片 URL（`<img>` / blob 拉取）自动附加 `?tenant_code=`（存储路径仍不含租户参数）
6. 后台「系统管理 → 租户管理」可列表 / 开户 / 启停（数据在平台库；需登录某个已有租户后操作）

未开启前端开关时不显示租户字段；若本地已有 `tenant_code` 仍会带 Header（后端 `off` 时忽略）。

运维命令补充：`tenant:list`、`tenant:enable` / `tenant:disable`、`tenant:seed-all [--class=...]`。

## 已废弃

- 共享表 + `WHERE tenant_id` / `ScopeTenant` / GlobalScope 行级方案（已移除相关文档与空壳代码）
- 「认证在平台库、业务在租户库」的折中中间态
- 用「无 tenant_id」误判「超级管理员」的 helpers
