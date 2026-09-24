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
TENANCY_MIGRATE_CONCURRENCY=2             # tenant:migrate-all / seed-all 并发；可用 --concurrency 覆盖；上限 100
TENANCY_BACKUP_CONCURRENCY=2              # 备份 fleet 并行 dump；上限 50
TENANT_BACKUP_SCHEDULE_MODE=full          # full=每日全舰队；rotate=按 BATCH 轮转
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
6. **异步运维**：单户 migrate / seed / backup / restore 走 `tenant_ops`（`long-running`）；生产需 Redis 队列 + long-running worker。`migrate-all` / `backup-all` 仍可用 CLI：`backup-all` 改为入队 long-running（分页扇出），定时 `backup-scheduled` 按天轮转 `TENANT_BACKUP_SCHEDULE_BATCH` 户。
7. **全舰队 migrate/seed 并发**：`TENANCY_MIGRATE_CONCURRENCY`（默认 2）控制 CLI `tenant:migrate-all` / `seed-all` **以及**平台 UI 批量 migrate/seed（`tenant_ops_fleet`）；CLI `--concurrency=N` 可临时覆盖。与 `QUEUE_LONG_RUNNING_CONCURRENT`（单户/backup）、`QUEUE_SCHEDULE_CONCURRENT`（采集）**不是同一旋钮**；见 [并发旋钮对照](#并发旋钮对照与队列一起调)。
8. **全舰队每日备份**：`TENANT_BACKUP_SCHEDULE_MODE=full`（默认）时 `tenant:backup-scheduled` 每天入队**全部** ready 租户；内部用 `tenant_ops_fleet` + `TENANCY_BACKUP_CONCURRENCY`（默认 2，上限 50）。`MODE=rotate` 仍按 `TENANT_BACKUP_SCHEDULE_BATCH` 轮转。CLI `backup-all` / UI 批量 backup 同路径。墙钟 ≈ ceil(户数/并发)×单户 dump 秒数；`QUEUE_LONG_RUNNING_CONCURRENT=1` 即可（fleet 占一槽）。



8. **CLI 运维记录**：	enant:migrate / migrate-all / seed / seed-all 会写入 	enant_op_logs（operator=cli，fleet 带同一 atch_id），平台可筛 last_op/last_op_status，并可一键「重试失败填充」。会话库名校验同时支持 MySQL DATABASE() 与 PostgreSQL current_database()。

## 公网部署（推荐）

1. **`TENANCY_RESOLVER=subdomain`**：租户以 `acme.example.com` 访问；apex/`www`/`platform` 等保留域**不接受** Header/Query 冒充（除非显式 `TENANCY_ALLOW_HEADER_FALLBACK=true`）。
2. 子域与 Header/body 冲突 → `tenant_hint_conflict`（400）。
3. **支付回调**：`POST /api/payment/notify/{type}/{tenant_code}`（渠道不会带租户 Header）；tenancy 开启时须带 `{tenant_code}`。
4. **连接池**：每租户 `TENANCY_POOL_MAX_*`（默认 idle 2 / open 20）；户多时按下方 [规模与推荐配置](#规模与推荐配置) 下调，避免打满 MySQL。
5. 平台控制台走 `platform.` 或独立域名；勿与租户子域混用。



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

以下为**经验起点**，需按监控回调，**不是硬性配额**。户多时首要瓶颈通常是**租户库连接池**（不是 Redis）。Redis 全站共用（缓存键经 `tenancy.CacheKey` 加 `t{id}:` 前缀），一般升规格即可；队列与缓存吵邻居时再考虑拆实例或分 DB。

### 出厂默认能支撑多少

「活跃」指**近期有流量、进程内已注册连接池的租户**，不是 `tenants` 表总行数。

| 场景 | 建议 |
|------|------|
| **默认池 + 单机部署** | 稳妥大约 **&lt; 50 活跃**；储备 **50–200**（建议已收紧 `OPEN`） |
| **调参 + Redis 队列 + API/Worker 分开** | **数百～约 1k 活跃** |
| **再加 `tenants.host` 分库主机 + `queue-schedule` Worker** | **几千活跃**（本产品目标量级） |

默认 `TENANCY_POOL_MAX_OPEN_CONNS=20`、`TENANCY_REGISTERED_MAX=0`（不限）。**几千商户不要原样上生产**，按下表调参。

**容量公式（租户库）：**

`进程内已注册的租户池数 × TENANCY_POOL_MAX_OPEN_CONNS × API实例数` ≪ 数据库 `max_connections`（预留平台库、备份、运维余量）。

同机多库时所有 `tenant_*` 仍计入**同一** MySQL/PG 实例的连接上限。

### 运行时双模式（少量 / 大量）

| 机制 | 配置 | 少量租户 | 大量租户 |
|------|------|----------|----------|
| 连接注册表空闲淘汰 | `TENANCY_REGISTERED_IDLE_TTL`（默认 900s） | 长时间无访问会释放池 | 避免注册池涨到总户数 |
| 注册池硬上限 | `TENANCY_REGISTERED_MAX`（默认 0=不限） | 可不设 | 建议 300–500 |
| 全舰队命令分页 | `TENANCY_SCOPE_AUTO_BATCH_AT`（200） / `TENANCY_SCOPE_AUTO_BATCH`（100） | 户数 ≤200 时一次跑完 | 超过阈值自动分页；分钟级任务带 rotate 游标 |
| 强制页大小 | `TENANCY_SCOPE_BATCH` | 0=跟自动策略 | 可强制每页 N 户 |
| 灵活定时 | `next_run_at` + tick 扇出 + `schedule` 队列（`QUEUE_SCHEDULE_CONCURRENT`） | 行为与按分钟匹配一致 | tick **只入队**；Worker 并行执行 handler |
| 全舰队备份 | `tenant:backup-all` / `backup-scheduled` | 小舰队可一次入队 | 入队 `long-running`；定时按天轮转 `TENANT_BACKUP_SCHEDULE_BATCH` |
| 全舰队 migrate/seed | `TENANCY_MIGRATE_CONCURRENCY` / `--concurrency` | 默认 2；单机同 host 建议 2–5 | 多 `tenants.host` 时可 20–50（真实并行 DDL/seed；勿盲目开满打爆 DB） |

### 全舰队 schema 迁移（`tenant:migrate-all`）

一户一库时每户独立 DDL/seed，墙钟大致为 `户数 × 单户耗时 / 并发`。`MigrateTenant` / `SeedTenant` 使用每 goroutine 绑定的 Schema+Orm（不翻转全局 `database.default`），故 `TENANCY_MIGRATE_CONCURRENCY` 对 migrate-all 与 seed-all 均为 **真正并行**。压力落在各租户库所在的 MySQL/PG 上。

| 部署 | 建议并发 | 说明 |
|------|----------|------|
| 单机 / 同 host 多库 | 2–5 | 默认 2；再高易顶 `max_connections` / 磁盘 IO |
| 约 100 户 / 一台租户库机，多 host | 20–50 | 开户填不同 `tenants.host` 分散；仍是**全局**并发上限，非按 host 分组 |
| 平台 UI 批量 migrate/seed | `TENANCY_MIGRATE_CONCURRENCY`（`tenant_ops_fleet`） | 与 CLI 同路径；`QUEUE_LONG_RUNNING_CONCURRENT=1` 即可 |

失败户会写 `provision_status=failed` + `last_migrate_error`，可筛失败后批量重试，不必每次全舰队重跑。发版：新镜像先平台 `migrate`，再 `tenant:migrate-all`（可 `--concurrency`），再切流量。

### 并发旋钮对照（与队列一起调）

这些参数**不是**同一个东西，调高其中一个不会自动提高另一个：

| 参数 | 控制什么 | 典型场景 |
|------|----------|----------|
| `TENANCY_MIGRATE_CONCURRENCY` | **本进程内** `tenant:migrate-all` / `tenant:seed-all` 同时处理几户（goroutine） | CLI 发版窗口全舰队 DDL/seed |
| `QUEUE_LONG_RUNNING_CONCURRENT` | **每个** long-running Worker 同时跑几个任务 | 单户 `tenant_ops`；**批量 migrate/seed/backup 只需 1**（fleet 占一槽） |
| `QUEUE_SCHEDULE_CONCURRENT` | **每个** schedule Worker 同时跑几个灵活定时/采集 job | 日常分钟级业务，与 migrate **抢租户库 IO** |
| `QUEUE_CONCURRENT` | default 队列并行度 | 导出、通知等普通任务 |

**一起调的原则（同机租户库）：**

1. **发版跑 `migrate-all` / `seed-all` 时**：把 `QUEUE_SCHEDULE_CONCURRENT` 临时降到 `1–2`（或先停 schedule tick），`QUEUE_LONG_RUNNING_CONCURRENT` 保持 `1`，**不要**同时在平台 UI 再点批量迁移——否则 CLI 并发 + Worker 并发叠在同一批 MySQL/PG 上。
2. **墙钟**：`≈ ceil(户数 / TENANCY_MIGRATE_CONCURRENCY) × 单户 migrate 耗时`；seed 通常远快于 migrate，可共用同一并发值。
3. **日常运行**：`TENANCY_MIGRATE_CONCURRENCY` 只影响 CLI，可长期保持较大；真正常驻压力看 `QUEUE_SCHEDULE_CONCURRENT × Worker 台数` 与 `TENANCY_POOL_MAX_OPEN_CONNS`。
4. **经验上界**：单机同 host 建议 CLI 并发 `2–5`；多 `tenants.host` 才开到 `20–50`。队列侧 long-running 默认 `1` 即可，schedule 按采集 SLA 单独加。
5. **批量 migrate/seed**：CLI 与平台 UI 批量都读 `TENANCY_MIGRATE_CONCURRENCY`。`QUEUE_LONG_RUNNING_CONCURRENT` 不必跟着开大。发版时仍勿同时 CLI + UI 两路叠压同一批库。

**平台 UI 批量 migrate / seed / backup：**

1. **批量** migrate/seed 入队 **一个** `tenant_ops_fleet` 任务，内部用 `TENANCY_MIGRATE_CONCURRENCY`（与 CLI `migrate-all` 同路径）。单户操作仍走普通 `tenant_ops`。`QUEUE_LONG_RUNNING_CONCURRENT` 对批量迁保持 **1** 即可（fleet 只占一个队列槽）。
2. Worker 机须跑 `queue-long-running`（勿设 `APP_DISABLED_RUNNERS=queue-*`）；API 机通常只 Dispatch。
3. 改 `.env` 后须 **重启消费 long-running 的 Worker 进程**（并发在启动时读入，见 `bootstrap/runners.go`）。只改 env 不重启 Worker → 仍是旧并发。只重启 API、Worker 不重启 → UI 入队照常但并行度不变。
4. 批量迁速度靠调大 `TENANCY_MIGRATE_CONCURRENCY`（改完 **下次** 入队的 fleet 即生效，或重启 Worker 前已入队的不受影响）。backup 批量同样走 `tenant_ops_fleet`，并行看 `TENANCY_BACKUP_CONCURRENCY`（`QUEUE_LONG_RUNNING_CONCURRENT=1` 即可）。

| 改哪个 | 要不要重启 | 说明 |
|--------|------------|------|
| `QUEUE_LONG_RUNNING_CONCURRENT` / `QUEUE_SCHEDULE_CONCURRENT` / `QUEUE_CONCURRENT` | **要**：重启对应队列 Worker | 启动时固定 Concurrent |
| `TENANCY_MIGRATE_CONCURRENCY` | CLI **下次** `artisan tenant:migrate-all` 即生效；长期跑的 API/Worker 不必为它重启 | 也可用一次性 `--concurrency=N`，无需改 env |
| 平台 UI 点迁移 | 不重启也能入队 | 并行度仍取决于已启动的 Worker 并发 |

**建议取值（可直接抄，同机一库主机）：**

| 场景 | `TENANCY_MIGRATE_CONCURRENCY`（CLI / UI 批量 migrate·seed） | `QUEUE_LONG_RUNNING_CONCURRENT`（单户 / backup 批量） | `QUEUE_SCHEDULE_CONCURRENT` |
|------|--------------------------------------|----------------------------------------|------------------------------|
| 本地 / 小舰队 &lt;50 户 | **2–3**（默认 2） | **1** | **5–10**（默认常 10） |
| 单机同 host，发版 migrate-all | **3–5** | **1**（发版窗口勿开大） | 临时 **1–2** 或停 tick |
| ~100 户/台库机，已分 `tenants.host` | **20–40** | **1–2** | **5–10** |
| 几千户、多 host | **40–50**（上限 100，慎用） | **1–2**；更快就加 Worker 台 | 按 SLA，常从 **10** 起 |

经验：UI 批量迁调 `TENANCY_MIGRATE_CONCURRENCY`（与 CLI 相同）；`QUEUE_LONG_RUNNING_CONCURRENT` 默认 **1** 即可。墙钟粗算 `ceil(户数/并发) × 单户 migrate 秒数`（本仓库实测单户 DDL 约十余秒量级）。

### 活跃量级 × 调参对照

| 活跃商户 | 租户池 | 队列 / 进程 | Redis |
|----------------|------|-------------|-------|
| &lt; 50 | 默认 `IDLE=2` / `OPEN=20` 可留；可先降到 `OPEN=10` | 可同机 Worker；`QUEUE_CONNECTION=redis` | 单机或小规格云 Redis |
| 50–200 | `IDLE=1–2`，`OPEN=3–5`；缩短 idle/lifetime（如 120 / 600） | API 与 Worker **分角色**（见 [生产清单](/deploy/production) §4.1）；`QUEUE_LONG_RUNNING_CONCURRENT` 保持较小 | 托管 Redis，盯 `used_memory` 与 backlog |
| 200–1000 | `OPEN=2–3`；`REGISTERED_MAX=300` 可选 | 独立 Worker：`default` + `long-running` + **`schedule`** | 更大规格；导出导入注意吵邻居 |
| 几千 | `OPEN=2`；`REGISTERED_MAX=300–500`；`tenants.host` **分库主机**；migrate 可 `TENANCY_MIGRATE_CONCURRENCY=20–50` | Worker 可水平扩；`QUEUE_SCHEDULE_CONCURRENT` 按采集耗时调 | 托管高可用；吵邻居再拆缓存/队列 |

### 机器规模与角色分布（经验）

同一二进制；用 `APP_DISABLED_RUNNERS` 区分角色（详见 [生产清单 §4.1](/deploy/production)）。以下 **vCPU / 内存** 为云主机粗算，按实际 QPS、采集耗时、导出频次加减。

| 活跃户量级 | API | Queue Worker | Scheduler | 平台 DB | 租户 DB | Redis |
|--------------|-----|--------------|-----------|---------|---------|-------|
| &lt; 50 | 1× 2vCPU 4GB（可同机跑 Worker） | 可合并 | 合并或单独 1 台 | 1×中小规格 | 可与平台同机多库 | 1×小规格 |
| 50–200 | 2× 2vCPU 4GB（只 HTTP） | 1× 2vCPU 4GB | **只 1 台** 开 schedule | 1×（提高`max_connections`） | 同机或开始拆 1 台租户库机 | 1×托管 |
| 200–1000 | 2–4× API | 2+× Worker（含 `schedule`） | 1 台 | 独立平台库 | **多台** `tenants.host` 分片 | 1×中大规格 HA |
| 几千 | 4+× API（LB） | 3+× Worker；`QUEUE_SCHEDULE_CONCURRENT`按需调 | 1 台 | 独立 + 备份 | 多台租户库（按 host 分配开户） | HA；吵邻居再拆 |

**角色要点：**

| 角色 | `APP_DISABLED_RUNNERS` | 必开队列 Runner |
|------|-------------------------|----------------|
| API | `queue-*` | 无（只 Dispatch） |
| Worker | 留空 | `queue-default`、`queue-long-running`、**`queue-schedule`**（灵活定时）、可选 `queue-search` |
| Scheduler | 可 `queue-*`（若不消费） | 只跑 `goravel:schedule`；**全站只 1 台** |

分钟级采集吞吐粗算：`并发槽 ≈ QUEUE_SCHEDULE_CONCURRENT × Worker 台数`。若 3000 户/分钟且每户耗时 5s，约需 **250** 并发槽（或降频）。

### 示例配置

约 100 活跃户、2 个 API：

```ini
TENANCY_POOL_MAX_IDLE_CONNS=1
TENANCY_POOL_MAX_OPEN_CONNS=5
TENANCY_POOL_CONN_MAX_IDLETIME=120
TENANCY_POOL_CONN_MAX_LIFETIME=600
TENANCY_REGISTERED_IDLE_TTL=900

CACHE_STORE=redis
QUEUE_CONNECTION=redis
QUEUE_CONCURRENT=2
QUEUE_LONG_RUNNING_CONCURRENT=1
QUEUE_SCHEDULE_CONCURRENT=5
# API 机：APP_DISABLED_RUNNERS=queue-*
```

几千活跃起点：

```ini
TENANCY_POOL_MAX_IDLE_CONNS=1
TENANCY_POOL_MAX_OPEN_CONNS=2
TENANCY_REGISTERED_IDLE_TTL=900
TENANCY_REGISTERED_MAX=400
TENANCY_SCOPE_AUTO_BATCH_AT=200
TENANCY_SCOPE_AUTO_BATCH=100
TENANCY_FLEX_SCHEDULE_TICK_LIMIT=2000
TENANCY_FLEX_SCHEDULE_QUEUE=schedule
TENANCY_MIGRATE_CONCURRENCY=40
QUEUE_SCHEDULE_CONCURRENT=10
TENANT_BACKUP_SCHEDULE_ENABLED=true
TENANT_BACKUP_SCHEDULE_BATCH=100
QUEUE_LONG_RUNNING_CONCURRENT=1
# 开户时填 tenants.host 分库主机；API 关 queue-* ；Worker 开 queue-schedule
```

**建议监控：** MySQL/PG `Threads_connected`、队列 pending / `queue:alert-backlog`、`schedule` / `long-running` 积压、Redis 内存与连接数。接近上限时先下调 `TENANCY_POOL_*` 或扩容 DB/Worker，**不要盲目加 API 副本**（副本会放大连接占用）。

## 对象存储与配额

- **默认共用盘**：全站一份 `FILESYSTEM_DISK`；路径用 `tenants/{code}/` 前缀隔离。租户后台**不可**改 `file_disk`（保存会被拒绝），仅可改导出格式。
- **可选自有桶（平台配置）**：租户 `storage_mode=custom` 时可填 s3/oss/cos/minio 凭证（密钥用 `APP_KEY` 加密）。新上传写入磁盘名 `tenant_byob_{id}`，并由附件记录该 disk。
- **切换**：shared ↔ custom **只影响新上传**。旧文件仍按 `attachments.disk` 读取。切回共享后请保留自有桶密钥，否则历史自定义文件可能无法访问。永久清理会删平台前缀，若仍有凭证也会清理自有桶前缀。
- **删除租户**：平台元数据**软删**入回收站；drop_database 仅 DROP 库/Schema；**此时不清理对象存储**。
- **回收站 / 永久删除**：GET /tenants?trashed=only；POST .../undelete 恢复；POST .../purge 可单独重试清理；DELETE .../force 默认异步清理对象+备份后再硬删并释放 code。保留 TENANCY_DELETED_RETENTION_DAYS；定时 tenant:cleanup-deleted。
- **存储限额** `storage_limit_bytes`（0=不限）：按租户库 `attachments.size` 汇总，上传前校验。


## 运维增强

1. **Landlord 迁移跳过**：平台表迁移（`tenants` / `platform_admins` / `jobs` / provision/migrate meta）在 `tenant_*` 连接上 `SkipOnTenantConnection` 空跑，避免污染租户库。
2. **Migrate / 运维可见性**：`last_migrate_error` / `migrated_at`；平台 UI 另有 `last_op` / `last_op_status` / `last_op_message` / `last_backup_path`。失败写 `provision_status=failed`（seed 失败不降级已 ready）。
3. **连接探测 / 异步操作**：`POST .../ping`；`.../migrate|seed|backup` 入队。
4. **登录限流**：`login` limiter 键含 body/query/header/`subdomain` 租户提示，避免跨租户互相锁号。
5. **日志**：带 `tenant_code` / `tenant_id` 前缀（`app/utils/logger`）。
6. **PG sslmode**：`TENANCY_POSTGRES_SSLMODE` 或 `DB_SSLMODE`。
7. **备份/恢复**：`tenant:backup [--keep=N]`、`tenant:backup-all`（入队，非同步 dump 全舰队）、`tenant:backup-scheduled`（日批轮转）；PG schema 隔离备份用 `pg_dump -n`，恢复用 `PGOPTIONS=--search_path`。
8. **CLI 范围**：`RunTenantScope` 仅遍历 **active + ready**；`tenant:migrate-all` 仍可覆盖 pending（单独查询）；并发见 `TENANCY_MIGRATE_CONCURRENCY` / `--concurrency`。
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
8. 浏览器跨域：`CORS_ALLOWED_HEADERS` 须含 `X-Tenant-ID`；子域靠 `TENANCY_BASE_DOMAIN` 通配；独立域优先同 Host 或演示站 `CORS_ALLOWED_ORIGINS` 空/`*`（见上文 CORS）；拆分 API 时建议 `ADMIN_ORIGIN_GUARD=true`（见 Origin 校验），勿为每户手改 CORS 白名单。
9. 订单搜索用 `search:*` / `SyncOrderSearch`（`SEARCH_*`），勿再接旧 ES outbox 链路。
10. IP 黑名单：进程内短 TTL（约 30s）缓存启用名单；CRUD 后立即失效。查库失败时在约 5 分钟内回退最近成功缓存，超时仍 **fail-closed**（503）。
11. 仅 `provision_status=ready` 的租户可绑定业务；HTTP 开户禁止同步 migrate，可走平台 UI 异步迁移或 CLI `tenant:migrate`。
12. 远程租户库禁止空账号回落平台 root；公网默认 `TENANCY_ALLOW_PLATFORM_DB_CREDENTIALS=false`。
13. 平台表迁移不得落在租户库（`SkipOnTenantConnection`）；migrate 失败须可在平台侧看到 `last_migrate_error`。
14. PG schema 隔离的 backup/restore 必须限定 schema；登录对 `tenant_not_ready` 返回 403（非 500）。
15. 公网优先 subdomain；支付回调必须带 `{type}/{tenant_code}` 路径。
16. 商户独立域名用边缘改写 Host（见上文）；勿为每个独立域改应用 env、CORS 白名单或独立部署。
17. 业务目录（`app/services` / `http` / `jobs` / `console`）禁止 `facades.Orm().Query()`；CI 跑 `bash scripts/check-tenant-orm.sh`（租户运维白名单除外）。

## Platform ops: domain board / onboard / health

- Tenant list: domain_status (unbound/pending/active/verify_failed) + domain_host search; health_status.
- Onboard API: POST /api/platform/tenants/onboard (create → queue migrate[+seed] → optional domain → login_links).
- Health: 	enant:health-inspect (hourly) writes health_* / last_ping_*; alerts via TENANT_OPS_ALERT_WEBHOOK_URL and optional TENANT_HEALTH_ALERT_MAIL.
- Manual: POST /api/platform/tenants/health-inspect.
