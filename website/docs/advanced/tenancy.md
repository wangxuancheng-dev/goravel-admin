# 多租户：一户一库 / Schema（MySQL + PostgreSQL）

默认 **`TENANCY_DRIVER=off`**：整站单库。

设为 **`database`** 后：每租户独立 database（MySQL）或 database/schema（PostgreSQL）；**租户认证与业务都在该租户库**。平台控制台使用独立账号与 `/api/platform`，不切租户库。

> 写业务模块只需读 [多租户：业务开发须知](/advanced/tenancy-dev)；生产调参见 [规模与推荐配置](/advanced/tenancy-scaling)。

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

### 连接管理（`app/tenantdb`）

- 租户连接由应用自己的注册表 `tenantdb.Default` 持有：用框架导出的构造器为每个连接单独构建（驱动 → gorm → Orm），**不写全局 config、不进框架的连接缓存**，何时建、换、关都由应用决定。
- 请求路径查找已注册连接只取读锁；新建或重建连接（含 ping，超时 5s）在锁外进行并按连接名去重，一个慢库或连不上的库不会拖住其他租户。
- `OrmQuery(ctx)` 按 ctx 中的连接名取连接；连接被淘汰后再次访问，会按 `tenants` 表自动重建。
- 每个租户请求都要读 `tenants` 行（识别租户、建连接、存储），所以缓存在共享缓存里（`app/tenantcache`，生产为 Redis，`TENANCY_TENANT_CACHE_TTL` 默认 60s）。失效是**自动**的：平台连接上任何成功写 `tenants` 表的语句（`Model(t).Update`、`Save`、`Delete`、`Where(...).Delete` 批量写，乃至应用内 `Exec("UPDATE tenants ...")`）都会**立即在所有节点失效**；能识别出租户的只失效该租户，识别不出的失效全部。停用、维护、删除不受缓存延迟影响，不需要手动调用任何失效函数。只有在应用之外手工改库（如 mysql 客户端）才会最多延迟一个 TTL。失效回调没挂上时缓存自动关闭。进程内另有一份本地副本，以带版本号的 key 存放，命中时每个请求只需从 Redis 读两个版本号；版本号一变本地副本就自然失效，不会读到旧数据。**多进程部署（API 与 worker 分开、多台机器）必须用共享缓存（`CACHE_STORE=redis`）**：`memory` 驱动只在本进程内失效，其他进程最多延迟一个 TTL。
- 迁移和 seeder 只能通过 `facades.Schema()` / `facades.Orm()` 访问数据库（不接收参数），所以 migrate / seed / `WithSchemaContext(ctx, fn)` 在**当前 goroutine** 上绑定一份独立的租户 Schema（`WithBoundTenantSchema`），不改任何全局状态，其他请求不受影响。后台协程一律用 `tenancyctx.Go(ctx, name, fn)` 启动（保留租户、去掉请求取消、兜底 panic），在其中用 `OrmQuery(ctx)`。

## 租户权益 / 模块套餐

> 仅登记到 platform_features 的模块走本页权益套餐；其余业务模块用角色菜单与权限控制。

### 在哪配置（平台控制台）

前提：`TENANCY_DRIVER=database`，用 **平台账号** 登录 `/platform/login`（不是租户后台）。

| 你要做的事 | 页面入口 | 说明 |
|------------|----------|------|
| 登记模块 / 手写模块 | **权益套餐** `/platform/entitlements` | 「登记模块」 |
| 登记模块内小功能 | 同上 | 「登记小功能」→ `module.{模块}.{能力}`，不绑菜单 |
| 新建套餐、改套餐里开哪些模块/额度 | 同上页下方 **套餐** → **套餐权益** | 每行 `key=value` |
| 给某租户订套餐 / 特批开关 | **租户** `/platform/tenants` → 打开详情 → **模块权益** | 分配套餐或 Switch 特批 |
| 代码生成时选是否纳入权益 | 租户后台 Dev → 代码生成器 | 勾选「纳入权益 / 始终开通 / 行数配额」 |

Landlord 迁移（只需跑平台库）：

```bash
go run . artisan migrate
```

### 手动配置模块套餐（推荐步骤）

1. **登记模块**（生成器 Install 过可跳过；手写模块必做）  
   - 打开 `/platform/entitlements` → **登记模块**（写入 `module.{name}`，可选行配额）  
   - 填 `module_name`（如 `guestbook`）、菜单 slug  
   - 可选：**始终开通**（不走套餐）、**行数配额**（套餐里再写额度）  
   - 手写模块还需：路由包 `middleware.EntitlementModule("guestbook")`；有行数配额则在 Create 调 `CheckAndConsumeModuleRows`

1b. **登记小功能**（可选，模块内能力如导出）  
   - 同页 → **登记小功能**：`module_name=member` + `capability=export` → `module.member.export`  
   - 不绑菜单；接口用 `middleware.EntitlementFeature("module.member.export")`；前端用 `/info` 的 entitlements 隐藏按钮  
   - 套餐权益里写 `module.member.export=true`

2. **配套餐权益**  
   - 同页新建套餐（如 `free` / `pro`）  
   - 点 **套餐权益**，按行填写，例如：

```text
module.guestbook=true
quota.module.guestbook.rows=1000
```

   - `false` = 该套餐不开此模块  
   - 行数 `-1` 或 `unlimited` = 不限；`0` = 不允许新建  

3. **订给租户**  
   - `/platform/tenants` → 详情 → **模块权益** → 选择套餐 → **分配套餐**  
   - 或对单个功能 Switch：**特批**（优先级高于套餐；「始终开通」的功能不可关）

4. **验证**  
   - 用该租户登录后台：未开通应无菜单，接口 403 `tenant_feature_disabled`  
   - 行数超限：创建 422 `entitlement_limit_exceeded`

### 关键 key 约定

| Key | 含义 |
|-----|------|
| `module.{name}` | 模块是否开通（boolean） |
| `module.{name}.{capability}` | 模块内小功能（boolean，无菜单） |
| `quota.module.{name}.rows` | 行数上限（limit，可选） |

解析优先级：`套餐默认 < 特批覆盖`；`always_on` 功能忽略套餐关闭。

### API（可选，等同于上面 UI）

- `GET /api/platform/features`、`POST /api/platform/features/register-module`、`POST /api/platform/features/register-capability`
- `GET/POST /api/platform/plans`、`PUT /api/platform/plans/{code}`（body 可带 `entitlements`）
- `GET /api/platform/tenants/{id}/entitlements`
- `POST /api/platform/tenants/{id}/subscription`
- `PUT/DELETE /api/platform/tenants/{id}/entitlements/{key}`

权益开关全局生效（管理端 / 用户端同一套 snapshot）。

### 商户侧：我的套餐

租户后台右上角头像菜单 → **我的套餐**（`/subscription`，仅 `TENANCY_DRIVER=database` 时显示），只读展示：

- 当前套餐、生效/到期时间（7 天内到期会有提醒）
- 额度使用情况（`quota.*` 的已用 / 上限）
- 功能清单：已开通 / 「升级后可用」

接口：`GET /api/admin/entitlements/me`（登录即可，不含特批原因等平台内部信息）。

未开通功能或额度超限时，接口返回 403 `tenant_feature_disabled` / 422 `entitlement_limit_exceeded`，前端统一弹出「请联系管理员升级套餐」的本地化提示。

### 开发接入（接口 + 页面）

权益与 RBAC 权限是两套：菜单/按钮权限用 `hasPermission`；套餐开关用 `hasEntitlement`。**超级管理员不会绕过权益**。

#### 1) 后端：模块路由整组拦截

```go
import "goravel/app/http/middleware"

// module.guestbook — 未开通整组 403 tenant_feature_disabled
router.Middleware(middleware.EntitlementModule("guestbook")).Group(func(router route.Router) {
    router.Resource("guestbooks", guestbookController)
})
```

#### 2) 后端：模块内小功能（单接口）

平台页「登记小功能」得到 `module.member.export` 后：

```go
router.Middleware(middleware.EntitlementFeature("module.member.export")).Get(
    "members/export", memberController.Export,
)
```

也可用 `middleware.EntitlementModule("member")` 包外层 + 小功能中间件包导出接口。

#### 3) 后端：行数配额（Create）

登记模块时勾选「行数配额」后，在创建 成功写入前：

```go
if err := services.ConsumeModuleRowsFromHTTP(ctx, "guestbook", 1); err != nil {
    return mapEntitlementError(ctx, err) // 422 entitlement_limit_exceeded
}
```

`TENANCY_DRIVER=off` 或未登记 `quota.module.*.rows` 时为 no-op。

#### 4) 前端：读 `/api/admin/info` 的 entitlements

登录后 `GET /api/admin/info` 在 tenancy 开启时返回：

```json
{
  "entitlements": {
    "version": 3,
    "plan_code": "pro",
    "channel": "admin",
    "features": { "module.guestbook": true, "module.member.export": false },
    "limits": { "quota.module.guestbook.rows": { "limit": 1000, "used": 12, "remaining": 988 } }
  }
}
```

React（`html-react`）：

```tsx
import { useUserStore } from '@/stores/user'

const canExport = useUserStore((s) => s.hasEntitlement('module.member.export'))
// ...
{canExport ? <Button onClick={onExport}>Export</Button> : null}
```

Vue（`html`）：

```js
import { useUserStore } from '@/store/user'
const userStore = useUserStore()
const canExport = userStore.hasEntitlement('module.member.export')
```

tenancy 关闭时 `hasEntitlement` 恒为 `true`（与后端中间件一致）。菜单隐藏由后端按 `menu_slugs` 过滤，小功能无菜单，必须前端自己藏按钮。


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
TENANCY_DB_DEDICATED_USER=false           # true：同机开户且未填账号时自动建专用 DB 账号（只授权本库），删库时一并删除
TENANCY_DB_USER_HOST=%                    # MySQL 专用账号的 host 部分；应用与库不同机时可收紧，如 10.0.%
TENANCY_POSTGRES_SSLMODE=                 # 空则回落 DB_SSLMODE / disable
TENANT_BACKUP_KEEP=10                     # tenant:backup 保留份数；0=不清理（旧名 TENANCY_BACKUP_KEEP 仍兼容）
TENANT_BACKUP_DISK=                       # 备份所在磁盘（s3/oss/cos/minio）；空=存于执行备份那台机器
TENANCY_MIGRATE_CONCURRENCY=2             # tenant:migrate-all / seed-all 并发；可用 --concurrency 覆盖；上限 100
TENANCY_BACKUP_CONCURRENCY=2              # 备份 fleet 并行 dump；上限 50
TENANT_BACKUP_SCHEDULE_MODE=full          # full=每日全舰队；rotate=按 BATCH 轮转
TENANCY_POOL_MAX_IDLE_CONNS=10
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

1. **平台连接钉死**：`PlatformOrmQuery` 使用 `tenancy.platform_connection`；租户 migrate / seed 只绑定当前 goroutine，不修改 `database.default` 等全局状态。
2. **生产 Artisan**：`APP_ENV=production` 白名单含 `tenant:*` / `platform:*`（开户/迁移可用）。
3. **连接回收**：`Forget` 立即关闭该租户的连接池；空闲（`TENANCY_REGISTERED_IDLE_TTL`）或超上限（`TENANCY_REGISTERED_MAX`）淘汰的连接在 2 分钟宽限期后关闭，再次访问时自动重建。
4. **开户状态**：HTTP/CLI 创建后为 `pending`；平台 UI 异步迁移或 CLI `tenant:migrate` → `migrating` → `ready`/`failed`；未 ready 禁止业务绑定。UI 入队后若 worker 未消费，约 30 分钟后允许重试（防永久卡住）。
5. **账号隔离**：远程库必须独立凭据；同机共用平台账号仅当 `TENANCY_ALLOW_PLATFORM_DB_CREDENTIALS=true`（**公网默认 false**）。没有自己账号的租户每次建连接都会检查该开关，关闭后立即无法再用平台账号连接（报 `tenant_credentials_required`），备份同样如此。推荐开启 `TENANCY_DB_DEDICATED_USER=true`：开户时用平台账号为租户创建 `t_<code>` 账号（随机密码，APP_KEY 加密保存），只授权该租户的库（MySQL `GRANT ALL ON tenant_x.*`；PostgreSQL 由该角色拥有库或 schema），删库时一并删除。这样即使代码写错，数据库服务器也会拒绝跨库访问。平台账号需要 `CREATE USER` + `GRANT OPTION`（MySQL）或 `CREATEROLE`（PostgreSQL）；PostgreSQL 共库 schema 模式下，建议对平台库执行 `REVOKE ALL ON SCHEMA public FROM PUBLIC`。
6. **异步运维**：单户 migrate / seed / backup / restore / rollback 走 `tenant_ops`（`long-running`）；生产需 Redis 队列 + long-running worker。`migrate-all` / `backup-all` / `rollback-all` 仍可用 CLI：`backup-all` 改为入队 long-running（分页扇出），定时 `backup-scheduled` 按天轮转 `TENANT_BACKUP_SCHEDULE_BATCH` 户。
7. **全舰队 migrate/seed 并发**：`TENANCY_MIGRATE_CONCURRENCY`（默认 2）控制 CLI `tenant:migrate-all` / `seed-all` / `rollback-all` **以及**平台 UI 批量 migrate/seed/rollback（`tenant_ops_fleet`）；CLI `--concurrency=N` 可临时覆盖。与 `QUEUE_LONG_RUNNING_CONCURRENT`（单户/backup）、`QUEUE_SCHEDULE_CONCURRENT`（采集）**不是同一旋钮**；见 [并发旋钮对照](/advanced/tenancy-scaling#并发旋钮对照与队列一起调)。
8. **全舰队 schema 回滚**：`tenant:rollback` / `tenant:rollback-all` 与平台 UI 单户/批量 rollback 共用 `TENANCY_MIGRATE_CONCURRENCY`（`tenant_ops_fleet`）。`--step=N` 回滚最近 N 个迁移文件；`--batch=N` 回滚 `migrations.batch=N`；二者为 0 时回滚上一批次。**首次 migrate 常把全部文件记为 batch 1**，默认 0/0 会整批撤销——生产请优先 `--step=1`，并先备份。
9. **全舰队每日备份**：`TENANT_BACKUP_SCHEDULE_MODE=full`（默认）时 `tenant:backup-scheduled` 每天入队**全部** ready 租户；内部用 `tenant_ops_fleet` + `TENANCY_BACKUP_CONCURRENCY`（默认 2，上限 50）。`MODE=rotate` 仍按 `TENANT_BACKUP_SCHEDULE_BATCH` 轮转。CLI `backup-all` / UI 批量 backup 同路径。墙钟 ≈ ceil(户数/并发)×单户 dump 秒数；`QUEUE_LONG_RUNNING_CONCURRENT=1` 即可（fleet 占一槽）。



10. **CLI 运维记录**：`tenant:migrate` / `migrate-all` / `seed` / `seed-all` / `rollback` / `rollback-all` 会写入 `tenant_op_logs`（operator=cli，fleet 带同一 `batch_id`），平台可筛 `last_op`/`last_op_status`，并可一键「重试失败填充」。会话库名校验同时支持 MySQL `DATABASE()` 与 PostgreSQL `current_database()`。

## 公网部署（推荐）

1. **`TENANCY_RESOLVER=subdomain`**：租户以 `acme.example.com` 访问；apex、`www.`、`platform.` **不接受** Header/Query 指定租户（除非显式 `TENANCY_ALLOW_HEADER_FALLBACK=true`）。`api.` 等 API 主机、IP、其他主机仍接受（前后端分离与公开 `<img ?tenant_code=>` 需要）；选中租户不等于能访问它，token 只在该租户自己的库里有效。
2. 子域与 Header/body 冲突 → `tenant_hint_conflict`（400）。
3. **支付回调**：`POST /api/payment/notify/{type}/{tenant_code}`（渠道不会带租户 Header）；tenancy 开启时须带 `{tenant_code}`。
4. **连接池**：每租户 `TENANCY_POOL_MAX_*`（默认 idle 10 / open 20）；户多时按下方 [规模与推荐配置](/advanced/tenancy-scaling) 下调，避免打满 MySQL。
5. 平台控制台走 `platform.` 或独立域名；勿与租户子域混用。
6. **`TENANCY_DB_DEDICATED_USER=true`**：每个租户一个只能访问自己库的 DB 账号，代码写错也跨不了库（见生产要点 5）。只影响此后新开的租户；存量租户继续用原账号，需要时在平台编辑租户凭据更换。



## 商户独立域名（tenant_domains）

子域 `{code}.${TENANCY_BASE_DOMAIN}` 默认可用。独立域名在**平台控制台**绑定，无需按户改 Nginx / 重启 API，也无需把每个租户域名写进 `CORS_ALLOWED_ORIGINS`。

| 场景 | 做法 |
|------|------|
| 大多数商户 | 泛解析 `*.example.com` + 子域 |
| 独立域（无 CDN） | `ssl_mode=edge`：CNAME 到 `TENANCY_DOMAIN_TARGET`，边缘 on-demand 出证 |
| 自有 CDN+SSL | `ssl_mode=customer_cdn`：CDN 回源，**回源 Host 保持客户域名** |
| Cloudflare Worker / Pages | `customer_cdn`：同一应用添加 Custom Domain；DNS **只需 TXT** 归属校验。勿橙云 CNAME 到 `TENANCY_DOMAIN_TARGET`（易 522） |

解析优先级：`active` 自定义 Host → Origin Host（拆分 SPA/API 时）→ 子域 → Header/Query。

公网启用独立域名时请设置 `TENANCY_BASE_DOMAIN`：子域解析仅认 `{code}.该主域`，避免把 `crm.客户域.com` 误当成租户短码。C 端 `/api/user`、`/api/public/*` 与后台共用 Host 绑定。前端构建请同步 `VITE_TENANCY_BASE_DOMAIN`，以便子域登录页隐藏租户码输入框。

### CORS（`CORS_ALLOWED_ORIGINS`）

| 设置 | 效果 |
|------|------|
| 空 / 仅有无效项 / `*` | 等价允许**任意 Origin**（演示站最省事；生产谨慎） |
| 明确白名单 | 自动追加 `http(s)://*.{TENANCY_BASE_DOMAIN}`；**不用**为每个子域手写 |
| 独立域 + 拆分 API | 优先：同 Host 反代 `/api`（无跨域）；演示可用空/`*`；或临时把该 Origin 写进白名单 |

应用中间件也会尝试放行 `tenant_domains` 中 `status=active` 的独立域 Origin；若预检仍被拦（例如框架 `rs/cors` 只认静态列表），用上表「空/`*`」或同 Host 即可，**不必**每加一个商户域就改 env。`CORS_ALLOWED_HEADERS` 须含 `X-Tenant-ID`（header 解析租户时）。

### Origin 校验（`ADMIN_ORIGIN_GUARD`）

仅多租户模式生效（`TENANCY_DRIVER` 非 `off`）。拆分部署（例如 Pages 前端 + `api.*`）时，`CORS_ALLOWED_ORIGINS` 常设为空/`*`以通过预检；启用 `ADMIN_ORIGIN_GUARD=true` 后，`/api/admin` 会校验浏览器 `Origin` Host：

| 来源 | 是否放行 |
|------|----------|
| `DOMAINS_ADMIN` 配置（含 `*.` 通配） | 是 |
| `TENANCY_BASE_DOMAIN` 及其子域 | 是 |
| `tenant_domains` 中 `status=active` 的独立域 | 是 |
| 无 `Origin`（curl / 原生客户端） | 是 |
| 其他未绑定域 | 否（403 `origin_not_allowed`） |

```ini
ADMIN_ORIGIN_GUARD=true
DOMAINS_ADMIN=admin.example.com,platform.example.com
TENANCY_BASE_DOMAIN=example.com
```

默认 `false`（不校验）；单租户（`TENANCY_DRIVER=off`）下自动跳过。与 Host 中间件 `DomainAllowTenantVanity` 配合：Host 管控请求打到哪个域，Origin 管控前端页面从哪来。


```ini
TENANCY_BASE_DOMAIN=example.com
TENANCY_DOMAIN_TARGET=tenants.example.com
TENANCY_DOMAIN_VERIFY_PREFIX=_goravel-tenant
TENANCY_DOMAIN_CACHE_TTL=60
```

### 运维：`TENANCY_DOMAIN_TARGET`（静态统一入口）

该配置是**一个固定主机名**，不是客户域名列表，也与平台控制台子域（如 `admin.example.com`）无关。

| 角色 | 做什么 |
|------|--------|
| 你（部署方） | 把 `tenants.example.com` **A/AAAA（或云厂商要求的记录）解析到边缘/反代/API 入口**，并让该入口按请求 **Host（客户域名）** 转发；edge 出证时调用 `GET /api/platform/public/tls-allow?host=` |
| 商户 | 在平台控制台绑定独立域名并完成 TXT 验证；`ssl_mode=edge` 时把 **客户域名 CNAME 到 `TENANCY_DOMAIN_TARGET`**（所有商户共用这一个目标） |
| 应用 | 查 `tenant_domains`（`status=active`）决定 Host 属于哪个租户；**不必**为每个客户域名改 env |

示例 DNS：

```text
# 你的静态入口（只配一次）
tenants.example.com.     A      203.0.113.10

# 商户各自配置（可有很多条，目标相同）
crm.customer-a.com.      CNAME  tenants.example.com.
shop.customer-b.com.     CNAME  tenants.example.com.
```

仅用子域、或独立域全部走 `customer_cdn` 时，可不设 `TENANCY_DOMAIN_TARGET`；一旦启用 edge 独立域，必须保证该主机名真实可达。

平台 API：`GET/POST /api/platform/tenants/{id}/domains`，`POST .../verify`，`PUT .../primary|disable`，`DELETE`；边缘询问 `GET /api/platform/public/tls-allow?host=`（仅 edge+active 返回 200）。

旧方案（Nginx 改写 Host 为 `{code}.主域`）仍可用；新产品请用 `tenant_domains`。
## 规模与推荐配置

已移至 [多租户：规模与推荐配置](/advanced/tenancy-scaling)（出厂容量、双模式、并发旋钮、机器分布、示例配置、多区域部署）。

## 对象存储与配额

- **默认共用盘**：全站一份 `FILESYSTEM_DISK`；路径用 `tenants/{code}/` 前缀隔离。租户后台**不可**改 `file_disk`（保存会被拒绝），仅可改导出格式。
- **可选自有桶（平台配置）**：租户 `storage_mode=custom` 时可填 s3/oss/cos/minio 凭证（密钥用 `APP_KEY` 加密）。新上传写入磁盘名 `tenant_byob_{id}`，并由附件记录该 disk。
  - 每个租户的存储驱动按其凭据单独构建并在进程内复用，空闲超过 `TENANCY_REGISTERED_IDLE_TTL` 或超过 `TENANCY_REGISTERED_MAX` 时按最久未用淘汰（与数据库连接共用这两个配置），下次用到自动重建；凭据**不写入全局 config**，其他租户和代码都读不到。在平台修改凭据后，下次访问自动换用新凭据，无需重启。
  - 代码里打开磁盘用 `utils.StorageDisk(disk)`；需要读取磁盘配置（如流式上传）用 `tenantstorage.DiskConfig(disk)`，不要直接读 `facades.Config()` 的 `filesystems.disks.tenant_byob_*`（那里没有）。
- **切换**：shared ↔ custom **只影响新上传**。旧文件仍按 `attachments.disk` 读取。切回共享后请保留自有桶密钥，否则历史自定义文件可能无法访问。永久清理会删平台前缀，若仍有凭证也会清理自有桶前缀。
- **删除租户**：平台元数据**软删**入回收站；drop_database 仅 DROP 库/Schema；**此时不清理对象存储**。
- **回收站 / 永久删除**：GET /tenants?trashed=only；POST .../undelete 恢复；POST .../purge 可单独重试清理；DELETE .../force 默认异步清理后再硬删并释放 code。永久删除（含定时 tenant:cleanup-deleted）一律清除搜索索引、对象存储、备份、数据库 / schema 与专用 DB 账号，任一步失败则保留在回收站、code 不释放，可重试。保留 TENANCY_DELETED_RETENTION_DAYS；定时 tenant:cleanup-deleted。
- **存储限额** `storage_limit_bytes`（0=不限）：按租户库 `attachments.size` 汇总，上传前校验。


## 运维增强

1. **Landlord 迁移跳过**：平台表迁移（`tenants` / `platform_admins` / `jobs` / provision/migrate meta）在 `tenant_*` 连接上 `SkipOnTenantConnection` 空跑，避免污染租户库。
2. **Migrate / 运维可见性**：`last_migrate_error` / `migrated_at`；平台 UI 另有 `last_op` / `last_op_status` / `last_op_message` / `last_backup_path`。失败写 `provision_status=failed`（seed 失败不降级已 ready）。
3. **连接探测 / 异步操作**：`POST .../ping`；`.../migrate|seed|backup|rollback` 入队。
4. **登录限流**：`login` limiter 键含 body/query/header/`subdomain` 租户提示，避免跨租户互相锁号。
5. **日志**：带 `tenant_code` / `tenant_id` 前缀（`app/utils/logger`）。
6. **PG sslmode**：`TENANCY_POSTGRES_SSLMODE` 或 `DB_SSLMODE`。
   - **跨驱动租户**：租户驱动与平台库不同（如平台 MySQL、租户 PostgreSQL）时必须填 host/port/账号，留空会被拒绝（空 host 表示建在平台库服务器上）。
   - **PG schema 隔离**：同库多个租户各占一个 schema；表存在判断、表清单只看本租户 schema（框架原生实现会扫全库所有 schema），可安全并行迁移。
   - 测试：设置 `TENANCY_TEST_PG_HOST` / `_PORT` / `_USER` / `_PASSWORD` 后 `TestPostgresTenantLifecycle` 会在平台库上创建 PG 租户（database + schema 隔离）跑全流程；CI 自带 postgres 服务。
7. **迁移回滚**：`tenant:rollback` / `rollback-all`（及 UI/fleet）；默认建议 `--step=1`，避免 0/0 整批撤销首次 migrate。
8. **备份/恢复**：`tenant:backup [--keep=N]`、`tenant:backup-all`（入队，非同步 dump 全舰队）、`tenant:backup-scheduled`（日批轮转）。
   - 存放位置：默认在执行备份那台机器的 `storage/backups/tenants/{code}/`。多实例部署时设 `TENANT_BACKUP_DISK`（`config/filesystems.go` 里的磁盘名，如 s3 / oss / cos / minio）：备份完成后上传到该磁盘的 `tenant-backups/{code}/` 并删除本地文件，列表、下载、恢复、清理、删除租户时的清理都走该磁盘，各实例看到同一份。也可以不配磁盘、把 `storage/backups` 挂到共享目录。
   - 框架的磁盘驱动在上传 / 下载时会把整个文件读进内存；特别大的租户库建议用共享目录。生产环境未配置或配置不完整时，启动日志会有 `[production]` 警告。
   - PG 只导出租户自己的 schema（`pg_dump -n`；独立库租户为 `public`），不会带上同库的其他 schema 租户。
   - PG 恢复在单个事务里先清空该 schema（`DROP SCHEMA ... CASCADE`，含备份后新建的分表）再重放 dump，失败整体回滚、租户保持原样；平台 UI 与 `tenant:restore` 走同一实现。
   - 需要与服务端同大版本或更新的 `pg_dump` / `psql`。
9. **CLI 范围**：`RunTenantScope` 仅遍历 **active + ready**；`tenant:migrate-all` 仍可覆盖 pending（单独查询）；并发见 `TENANCY_MIGRATE_CONCURRENCY` / `--concurrency`。
10. **未绑定隔离**：tenancy 开启但 ctx 未绑定时，`CacheKey` → `t_unbound:*`，`StoragePrefix` → `tenants/_unbound_/`（不与共享根冲突）。

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
go run . artisan tenant:rollback {id|code} [--step=N] [--batch=N]
go run . artisan tenant:rollback-all [--step=N] [--batch=N] [--concurrency=N]
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

### 清理孤儿库（升级后执行一次）

以前的永久删除只删租户行，库和专用账号会留在服务器上，导致同一 code 再开户时被 `tenant_storage_exists` 拒绝。用下面的命令查找并清理：

```bash
go run . artisan tenant:orphans          # 只列出：带租户前缀、但没有任何租户（含回收站）引用的库 / schema
go run . artisan tenant:orphans --drop   # 确认后删除（含配对的 t_<code> 账号），不可恢复，先备份
```

只认带 `TENANCY_DATABASE_PREFIX` / `TENANCY_SCHEMA_PREFIX` 前缀的库，平台库和保留 schema 永远不会被删除。没有配对孤儿库的 `t_` 账号只列出、不删除（可能是 DBA 自建的）。

### 建库与密码

- `host` 空：在平台实例 `CREATE`
- `host` 有值：连目标主机系统库再 `CREATE`
- `--skip-create` / `skip_create`：库已存在，只登记
- `tenants.password`：仅 `enc:v1:` + `APP_KEY` 密文；无明文兼容
- HTTP `POST /api/platform/tenants`：**禁止** `migrate=true`（返回 `tenant_migrate_via_cli`）

## 三类日志（勿混用）

| 类型 | 存哪 | 回答什么 | 哪里看 |
|------|------|----------|--------|
| 操作日志 | 租户库 / 平台 `platform_operation_logs` | 谁改了什么（审计） | 租户后台 / 平台操作日志 |
| 租户运维日志 | 平台 `tenant_op_logs` | migrate/seed/backup 成没成 | 平台运维执行记录 |
| 系统日志 | 各连接库 `system_logs` | panic、库失败、基础设施报错 | 租户后台；平台系统日志（只读，可选 `tenant_code`） |

无租户上下文的平台进程错误经 `SystemLogOrmQuery` 写入 landlord `system_logs`；租户内错误仍落在租户库，**不**双写到平台库。

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
| GET | `/api/platform/login-logs` | Platform login logs (landlord) |
| GET/POST | `/api/platform/admins` | Platform admins list / create (owner write) |
| GET/PUT/DELETE | `/api/platform/admins/{id}` | Platform admin detail / update / soft-delete |
| POST | `/api/platform/admins/{id}/reset-password` | Reset platform admin password |
| GET | `/api/platform/login-logs/{id}` | Platform login log detail |
| GET | `/api/platform/operation-logs` | Platform operation logs (landlord) |
| GET | `/api/platform/operation-logs/{id}` | Platform operation log detail |
| GET | `/api/platform/system-logs` | 只读系统日志（默认平台库；`tenant_code` 切到该租户库） |
| GET | `/api/platform/system-logs/module-options` | 当前范围内的 module 筛选项 |
| GET | `/api/platform/system-logs/{id}` | 系统日志详情（范围同列表） |
| GET | `/api/platform/tenants/{id}/system-log-summary` | 近 24h error/warning 计数 + 最近 error |
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
| `OrmQuery(ctx)` / `OrmTransaction(ctx, fn)` | 租户业务默认入口；多租户开启时 ctx 为 nil 或没绑定租户直接报 `ErrNoTenantContext`（迁移 / seed 的 goroutine 绑定除外） |
| `PlatformOrmQuery(ctx)` / `PlatformQuery()` | 平台数据（有 ctx / 无 ctx）；禁止 `PlatformOrmQuery(nil)` |
| `tenancyctx.Go(ctx, name, fn)` | 启动后台协程：保留租户、去掉请求取消、兜底 panic |
| `tenancy.CacheKey` / `StoragePrefix` | 缓存键 / 对象存储路径前缀 |
| `models.PlatformModel` | 平台表模型标记；租户连接拒绝访问这些表（`tenantdb.ErrPlatformTable`） |
| `TenantConnectionService.TenantContext(t)` | 平台代码读写某个租户的数据：拿到绑定该租户的 ctx |
| `RunTenantScope` / `--tenant` | CLI 按租户执行 |
| `SchemaHasTable` | 按 ctx 查租户表是否存在 |
| `WithSchemaContext(ctx, fn)` | 在当前 goroutine 上以 ctx 所属租户运行 `facades.Orm()` / `facades.Schema()`（仅迁移式代码、seeder 这类无法接收 ctx 的代码） |

### 硬性规则

1. 业务用 `OrmQuery(ctx)`；平台路由**不**挂 `Tenant` 中间件；平台数据用 `PlatformOrmQuery(ctx)` / `PlatformQuery()`（钉死平台连接）。规则说明见 [业务开发须知](/advanced/tenancy-dev)。
2. 隔离是**切库/切 Schema**，不是行级 `tenant_id` GlobalScope。
3. 缓存/上传（含 `chunks/`、导出、导入与附件临时目录）走租户前缀。
4. 搜索索引绑定后为 `{code}_orders`（完整 code，不截断）；未绑定 fail-closed，禁止回退共享 `orders`。租户彻底删除（平台 force-delete / `tenant:cleanup-deleted` / purge 后删除）前先删除其全部搜索索引，删除失败则保留租户行、code 不释放（避免新租户复用 code 时读到旧租户文档）；创建租户时也会先清理同名残留索引。
5. 导出/搜索队列缺 `tenant_id` 时 fail-closed，禁止写到平台库。
6. HTTP 手动跑定时任务自动带当前 `--tenant`，禁止扫全租户；cron 可遍历启用租户。
7. 队列表 `jobs` / `failed_jobs` 读平台连接（`QUEUE_DATABASE_CONNECTION`）。
8. 浏览器跨域：`CORS_ALLOWED_HEADERS` 须含 `X-Tenant-ID`；子域靠 `TENANCY_BASE_DOMAIN` 通配；独立域优先同 Host 或演示站 `CORS_ALLOWED_ORIGINS` 空/`*`（见上文 CORS）；拆分 API 时建议 `ADMIN_ORIGIN_GUARD=true`（见 Origin 校验），勿为每户手改 CORS 白名单。
9. 订单搜索用 `search:*` / `SyncOrderSearch`（`SEARCH_*`），勿再接旧 ES outbox 链路。
10. IP 黑名单：进程内短 TTL（约 30s）缓存启用名单；CRUD 后立即失效。查库失败时在约 5 分钟内回退最近成功缓存，超时仍 **fail-closed**（503）。
11. 仅 `provision_status=ready` 的租户可绑定业务；HTTP 开户禁止同步 migrate，可走平台 UI 异步迁移或 CLI `tenant:migrate`。
12. 远程租户库禁止空账号回落平台 root；公网默认 `TENANCY_ALLOW_PLATFORM_DB_CREDENTIALS=false`。
13. 平台表迁移不得落在租户库（`SkipOnTenantConnection`）；migrate 失败须可在平台侧看到 `last_migrate_error`。
14. PG schema 隔离的 backup/restore 必须限定 schema；登录对 `tenant_not_ready` 返回 403（非 500）。
15. 公网优先 subdomain；支付回调必须带 `{type}/{tenant_code}` 路径。
16. 商户独立域名用边缘改写 Host（见上文）；勿为每个独立域改应用 env、CORS 白名单或独立部署。
17. CI 跑 `go run ./scripts/tenantlint`，按调用点检查四条规则：`orm`（`app/` 下禁止 `facades.Orm().Query/Transaction/WithContext/Connection`）、`nilctx`（禁止 `OrmQuery(nil)` / `PlatformOrmQuery(nil)` 等）、`cache`（`facades.Cache()` 所在函数须经 `tenancy.CacheKey`，可通过同包的 key 辅助函数）、`goctx`（`go` 语句不得捕获请求 ctx，改用 `tenancyctx.Go`）。确属例外的，在调用处或函数注释里写 `//tenantlint:allow <rule>: <原因>`。迁移（`database/migrations`）里的数据操作用 `migrationQuery()`：它在迁移事务内，`facades.Orm()` 不在（PostgreSQL 下看不到本迁移刚建的表）。
18. 平台表（`models.PlatformModel`）与 landlord-only 迁移建的表必须一致（`TestPlatformTablesMatchLandlordOnlyMigrations`）；租户连接对这些表的 ORM 读写一律拒绝。
19. `TestEveryAdminRouteIsTenantIsolated` 在租户 A 写入标记，再以租户 B 调用全部 GET 接口，断言拿不到 A 的数据；新增接口自动纳入。`TestTenantResolutionSecurity` 覆盖子域与 Header 冲突、公网 apex 拒收客户端租户、平台 / 租户 token 不互通、停用 / 维护立即生效、复用 code 后旧 token 失效。
20. **code 复用不继承任何东西**：永久删除先清掉库、存储、备份与索引才释放 code；开户遇到同名数据库 / schema 一律拒绝（`tenant_storage_exists`），不会接管；DBA 预建的空库用 `skip_create` 接管。开户失败直接物理删除租户行（并清掉本次建出的库与账号），code 立即可重用。
21. **租户解析的状态码**：缺少租户标识 400；租户不存在 404；租户停用或尚未 ready 403；维护中 503；租户库或平台库暂时连不上 **503 + `Retry-After: 5`**（`tenant_connection_failed` / `tenant_lookup_failed`），平台库故障不会被报成 404；缺少凭据之类的配置问题 500，并返回各自的错误码。探活和 CDN 可以据此区分「租户不存在」和「库挂了」。

## Platform ops: domain board / onboard / health

- Tenant list: domain_status (unbound/pending/active/verify_failed) + domain_host search; health_status.
- Onboard API: POST /api/platform/tenants/onboard (create → queue migrate[+seed] → optional domain → login_links).
- Health: tenant:health-inspect (hourly) writes health_* / last_ping_*; alerts via TENANT_OPS_ALERT_WEBHOOK_URL and optional TENANT_HEALTH_ALERT_MAIL.
- Manual: POST /api/platform/tenants/health-inspect.
