# 开源定位与生产配置

## 0. 建议阅读路径（先窄后宽）

1. [快速开始](/guide/getting-started) — 先跑起来（默认单库）
2. 本页 — 模块边界与**能力分档**（不必一次全开）
3. [系统架构](/guide/architecture) → [开发指南](/guide/development) / [代码生成器](/guide/code-generator) / [扩展包示例 feedback](/guide/package-example)
4. 上线：[生产清单](/deploy/production)（含[最小生产默认路径](/deploy/production#10-最小生产默认路径一户一库)）→ [SaaS 核对清单](/advanced/saas)
5. 写租户业务时再读 [多租户：业务开发须知](/advanced/tenancy-dev)；户多了再读 [规模与推荐配置](/advanced/tenancy-scaling)

**前端：** 新功能以 React（`html-react/`）为准，Vue（`html/`）跟进对齐，见 [双前端对齐](/guide/frontend-parity)。

---

## 1. 适用 / 不适用

### 适合

- 中小多租户 SaaS 运维（一户一库、平台控制台、开户迁移门禁）
- 企业内部后台、运营后台、管理中台
- Goravel + Vue / React 的二次开发脚手架
- 需要 RBAC、菜单、日志、导出、代码生成器的管理端
- 中小规模业务扩展（用户、订单、支付管理等示例能力可选用）

### 不适合

- 金融级交易核心、强一致支付中台（支付模块为管理端 + 网关示例）
- 超大规模、多区域、强 SLA 的商业 SaaS 产品中台
- 未做运维规划就开启「分表 + ES」当生产核心

演示站账号仅用于体验；生产请改默认管理员密码，并配置独立密钥。

---

## 1.1 支付模块边界

| 能力 | 状态 |
|------|------|
| 支付方式 CRUD、支付记录列表/详情/导出 | ✅ 可用 |
| 为订单创建支付单 `POST /api/admin/payments` + 可选 `initiate` | ✅ 参考下单 |
| **Mock 网关** 下单 / 查询 / 回调 → `ApplyPaidResult`（支付+订单幂等已支付） | ✅ 本地可跑通，见 [支付参考](/advanced/payments) |
| 微信 / 支付宝下单客户端调用（gopay） | ⚠️ 示例代码，需自备商户配置 |
| 微信 / 支付宝查询与回调验签 | ⚠️ 返回 `payment_gateway_not_implemented`（501）；验签后复用 `ApplyPaidResult` |
| 新渠道扩展 | ✅ `app/payment/gateways` + `RegisterGateway` + 通用 `notify/{type}`，见 [支付参考](/advanced/payments) §6 |
| 退款 API / 原路退 | ❌ 未提供（余额日志里的 refund 类型仅统计用） |

公网请用 `PAYMENT_GATEWAYS_ENABLED` 限制网关，并勿给无关角色支付权限。

---

## 2. 能力分档（不删功能，分层暴露）

能力都可以留着；第一次上线只盯 **A 档**。B/C 档需要时再展开，避免把「选项密度」当成「人人必背」。

### A. 默认必开（公网 / 一户一库）

| 项 | 说明 |
|----|------|
| 认证授权、RBAC、菜单、日志 | 管理端核心 |
| Redis 缓存 + 队列 | 生产强制（`CACHE_STORE` / `QUEUE_CONNECTION`） |
| 一户一库最小路径 | `subdomain` + 专用库账号 + TLS + 可信代理；见 [最小生产默认路径](/deploy/production#10-最小生产默认路径一户一库) |
| API / Worker 分角色 | `long-running` Worker；见生产清单 §4.1 |

单库演示（`TENANCY_DRIVER=off`）可先不看租户行；本地可用 sync/memory。

### B. 按规模再开

户数上去后再调；细节在 [规模与推荐配置](/advanced/tenancy-scaling)：

- 连接池 `TENANCY_POOL_*`、`TENANCY_REGISTERED_MAX`
- 舰队并发 `TENANCY_MIGRATE_CONCURRENCY` / `TENANCY_BACKUP_CONCURRENCY`
- 备份磁盘 `TENANT_BACKUP_DISK`、日程 `TENANT_BACKUP_SCHEDULE_*`
- 队列并发 `QUEUE_LONG_RUNNING_CONCURRENT` / `QUEUE_SCHEDULE_CONCURRENT`

**中小部署（约 50 活跃户以下）通常不必先懂这些旋钮。**

### C. 按业务选装（默认关或演示向）

| 模块 | 何时需要 | 文档 |
|------|----------|------|
| 搜索（ES / Meili） | 订单/全文检索 | [搜索](/advanced/search) |
| 分表 | 单表过大、按月归档 | [分表](/advanced/sharding) |
| 套餐 / entitlement 强制 | 真按 plan 限功能或行数 | [多租户 · 套餐](/advanced/tenancy) |
| 商户独立域名 | 白牌 / 自定义域 | [多租户 · 独立域名](/advanced/tenancy) |
| 支付网关真实验签 | 自备商户号后扩展 | [支付](/advanced/payments) |
| AI Lab / pprof / Swagger | 开发与排障 | 生产关闭或限权 |
| 调度演示 | 样板，非营销中台 | [调度演示](/advanced/scheduled-demos) |
| OpenTelemetry | 统一观测 | `OTEL_*` |

**AI：** 未配置 `AI_API_KEY` 时相关菜单隐藏。详见 `.env.example`。

### 模块开关

| 变量 | 默认 | 说明 |
|------|------|------|
| `PAYMENT_GATEWAYS_ENABLED` | （空） | 启用渠道白名单，如 `wechat,alipay`；空 / `*` / `all` = 全部已注册。**生产建议显式白名单**；新渠道见 [支付参考](/advanced/payments) §6 |
| `APP_ENABLE_DEV_TOOL` | `false` | 生产需显式 `true` 才开放开发工具。表单演示：`local/development/test` 默认可见；代码生成器：仅 `local/development` 默认可见（`test` 默认隐藏） |

登录 `Info` 下发 `payment_gateways`（网关白名单）；业务菜单可见性由角色菜单与权限控制。

### 数据权限（租户内行级）

挂在**角色**上（`roles.data_scope`），多角色取最宽；`super-admin` 始终全部数据。与一户一库租户隔离正交：先租户连接，再行级过滤。

| 值 | 含义 |
|----|------|
| 1 | 全部数据 |
| 2 | 自定义部门（`role_department`） |
| 3 | 本部门 |
| 4 | 本部门及以下 |
| 5 | 仅本人 |

已接入列表：管理员（`department_id`）、文章 / 附件 / 导出记录（`admin_id`）。服务入口：`ApplyDataScope` / `ResolveAdminDataScope`。

### 业务字段脱敏（列表 / 导出）

对配置的敏感字段在**列表**与**导出**中掩码；**详情 Show 保持明文**，避免编辑表单把掩码写回库。绕过角色（默认 `super-admin`）看明文。

**按租户 / 站点：**后台 → 系统管理 → 配置管理 → 字段脱敏，写入当前库 `configs`（group=`desensitize`）。`TENANCY_DRIVER=database` 时每个租户库独立——只有需要脱敏的租户打开即可；未配置时回退 `.env` 的 `DESENSITIZE_ENABLED`。

| 配置 | 默认 | 说明 |
|------|------|------|
| `DESENSITIZE_ENABLED` | `true` | 进程默认（`config/desensitize.go`） |
| `configs.desensitize.enabled` |（env）| 租户/单站覆盖 |
| bypass slug（env 或 DB） | `super-admin` | 逗号分隔 |
| modules | `user` / `admin` 的 `phone`、`email` | 策略：`phone` / `email` / `hide` / `keep_ends` |

已接入：用户列表与用户导出、管理员列表与管理员同步导出。Update 会忽略已含 `*` 的掩码值。

---

## 3. 最小生产配置

适用于：后台管理为主、暂不分表、暂不用 ES。

```ini
APP_ENV=production
APP_DEBUG=false
APP_KEY=          # 必填：go run . artisan key:generate
JWT_SECRET=       # 必填：强随机字符串

LOG_CHANNEL=stack
LOG_LEVEL=info

DB_CONNECTION=mysql
# ... 生产库连接 ...

CACHE_STORE=redis
QUEUE_CONNECTION=redis
QUEUE_CONCURRENT=2
QUEUE_TRIES=5

# DOMAINS_ADMIN=admin.example.com

# 生产默认关闭
SWAGGER_ENABLED=false
# APP_DISABLED_RUNNERS=  # Worker 机勿设 queue-*；API 多机见生产清单 §4.1
```

**上线检查（最小）：**

1. `migrate` 成功，`db:seed` 后修改默认 `admin` 密码  
2. Redis 可用；Web 与 Queue Worker 常驻（多机分角色配置见 [生产清单](/deploy/production) §4.1）  
3. HTTPS + 反向代理  
4. 关闭或限权：Swagger、pprof、代码生成器  
5. 日志磁盘与备份策略就绪  
6. 使用 `.env.production.example` 起步，**勿**直接用 `docker-compose.yml` 默认口令上生产  

部署细节见 [生产清单](/deploy/production)（健康检查 `/health` `/ready`、任务中心/导入权限、队列积压告警、上线清单）、[编译与部署](/deploy/build)、[Docker 生产部署](/deploy/docker)。升级含任务中心时务必按生产清单 **§1.1** 核对 `export/TaskCenter` 菜单与 `import.*` 权限。

### 资源归属（管理端）

- **导出**：下载 / 进度 SSE / 删除仅本人或配置的 `admin.super_admin_id`  
- **附件**：私有文件读/写同归属规则；公开附件（`is_public=1`）已登录管理员可读，改删仍需所有者或超管  
- **支付**：mock 可跑通下单/回调/订单同步；微信/支付宝查询与回调验签为 `payment_gateway_not_implemented`（501），见 [支付参考](/advanced/payments)

---

## 4. 进阶配置

按需叠加：

### 4.1 异步导出 / 长任务

```ini
QUEUE_CONNECTION=redis
QUEUE_LONG_RUNNING_CONCURRENT=1
# Worker 需消费 long-running 队列（见 bootstrap runners）
```

### 4.2 分表

```ini
# SHARDING_TIME_SUFFIX_LAYOUT=200601
# SHARDING_MAX_TIME_RANGE_MONTHS=3
# SHARDING_ID_LOOKUP_SCAN_MONTHS=6
# SHARDING_USER_BALANCE_LOGS_SHARDS=4
```

并配置定时任务创建未来分表（见分表文档）。

### 4.3 搜索引擎（Elasticsearch / Meilisearch）

详见 [搜索](/advanced/search)。

```ini
SEARCH_DRIVER=elasticsearch   # 或 meilisearch
SEARCH_ENABLED=true
SEARCH_SYNC_ORDERS=true
SEARCH_QUEUE=search
SEARCH_SYNC_WORKER=auto
# SEARCH_OUTBOX_ENABLED=true
ELASTICSEARCH_URLS=http://127.0.0.1:9200
# MEILISEARCH_HOST=http://127.0.0.1:7700
```

需要搜索集群 + `queue-search` Worker。Outbox 积压可用 `go run . artisan search:retry-outbox` 补偿。初始化：`search:init-orders-index`，全量：`search:sync-orders`。其他业务索引用 `search.RegisterDefinition` 扩展（见 [搜索](/advanced/search)）。

### 4.4 OpenTelemetry

```ini
# OTEL_TRACES_EXPORTER=otlptrace
# OTEL_METRICS_EXPORTER=otlpmetric
# OTEL_EXPORTER_OTLP_TRACES_ENDPOINT=http://127.0.0.1:4318
```

未配置 exporter 时框架会自动禁用 `goravel:telemetry` runner。

---

## 5. 相关文档

| 文档 | 说明 |
|------|------|
| [快速开始](/guide/getting-started) | 本机开发或 Docker 一键跑通 |
| [编译与部署](/deploy/build) | 编译与部署 |
| [测试指南](/guide/testing) | 测试指南 |
| [分表迁移](/advanced/sharding-migration) | 分表 |
| [错误码](/reference/error-codes) | 错误码 |
| [系统架构](/guide/architecture) | 架构 |
| [多租户](/advanced/tenancy) | 一户一库 / PG Schema 多租户（`TENANCY_DRIVER=database`） |
| [搜索](/advanced/search) | Elasticsearch / Meilisearch 切换、订单同步、文章扩展 |
| [生产清单](/deploy/production) | 生产上线清单、`/health` `/ready`、告警 |
| [SaaS 核对清单](/advanced/saas) | 中小 SaaS 上线核对清单 |
| [支付参考](/advanced/payments) | 订单/支付参考链路、mock 与微信支付宝扩展点 |
