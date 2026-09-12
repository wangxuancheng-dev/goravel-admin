# 开源定位与生产配置

本文档面向把本项目当作**开源后台管理系统 / 二次开发底座**的使用者。  
目标不是对标商业 SaaS 全套 SRE，而是：**能跑起来、能放心二次开发、能按需上生产。**

---

## 1. 适用 / 不适用

### 适合

- 企业内部后台、运营后台、管理中台
- Goravel + Vue / React 的二次开发脚手架
- 需要 RBAC、菜单、日志、导出、代码生成器的管理端
- 中小规模业务扩展（用户、订单、支付管理等示例能力可选用）

### 不适合（至少不能「开箱当核心」）

- 金融级交易核心、**强一致支付中台**（本仓库支付模块是后台管理 + 网关示例骨架）
- 超大规模、多区域、强 SLA 的商业 SaaS 产品中台
- 未做运维规划就直接开启「分表 + ES + 多队列」当生产核心

> 演示站账号仅用于体验，**切勿用于生产**。生产请改默认管理员密码，并配置独立密钥。

---

## 1.1 支付模块边界（必读）

后台「支付方式 / 支付记录」用于**管理端演示与二次开发参考**，当前能力与缺口如下：

| 能力 | 状态 |
|------|------|
| 支付方式 CRUD、支付记录列表/详情/导出 | ✅ 可用 |
| 微信 / 支付宝下单客户端调用（gopay） | ⚠️ 示例代码，需自备商户配置 |
| 支付结果查询 `POST /api/admin/payments/{id}/query` | ⚠️ 路由已接好，网关返回 `payment_gateway_not_implemented`（501） |
| 支付回调 `POST /api/payment/notify/{wechat\|alipay}[/{tenant}]` | ⚠️ 公开路由已接好；tenancy 开启须带 `{tenant}`；网关返回 `payment_gateway_not_implemented`（501）；**未做验签/落库** |
| 退款 API / 原路退 | ❌ 未提供（余额日志里的 refund 类型仅统计用） |

**结论：** 不要把本项目默认当成可上线的收单 / 清算系统。**默认 `MODULE_PAYMENTS_ENABLED=false`（demo-only）**；需要演示后台支付管理 UI 时再显式打开。回调/查询路由仅便于二次开发对接，不等于可用网关。

---

## 2. 模块分层：核心 vs 进阶

### 核心（默认可跑）

| 能力 | 说明 |
|------|------|
| 认证授权 | JWT、RBAC、菜单权限 |
| 系统管理 | 管理员、角色、部门、字典、配置 |
| 日志 | 操作日志、登录日志、系统日志 |
| 基础导出 | 列表导出（小数据可同步；异步导出见进阶） |
| 代码生成器 | 本地/开发环境使用；生产请限权或关闭 |

**最小依赖：** MySQL（或兼容库）+ 可运行的 Go 服务。  
本地开发可用 `QUEUE_CONNECTION=sync`、`CACHE_STORE=memory`（不推荐生产）。

### 进阶（可选，按需开启）

| 模块 | 何时需要 | 相关配置 / 文档 |
|------|----------|-----------------|
| Redis 缓存 / 队列 | 生产导出、异步任务、限流与锁 | `CACHE_STORE`、`QUEUE_CONNECTION` |
| 订单 / 支付分表 | 数据量大、按月归档 | [SHARDING_MIGRATION.md](./SHARDING_MIGRATION.md)、`SHARDING_*` |
| Elasticsearch | 订单检索、全文检索 | `ELASTICSEARCH_*`、ES Worker |
| 多队列驱动 | Kafka / RabbitMQ / NSQ / Redis Stream | `.env.example` 队列段 |
| OpenTelemetry | Jaeger / Grafana 等统一观测 | `OTEL_*`、[Telemetry 文档](https://www.goravel.dev/zh_CN/digging-deeper/telemetry.html) |
| AI / pprof / Swagger | 开发与排障 | 生产默认关闭或限权 |
| 一户一库多租户 | 大商户隔离（默认关闭） | `TENANCY_DRIVER=database`、平台 `/api/platform`、见 [TENANT_RESERVED.md](./TENANT_RESERVED.md) |

**AI（可选）：** 用于「代码生成器 → AI 辅助」与顶级 **AI 实验室**（文本 / 视觉 / 图片 / 语音 SDK 演示，演示站可用）。未配置 `AI_API_KEY`（或兼容别名 `OPENAI_API_KEY`）时，相关菜单与标签页自动隐藏。AI 实验室按管理员账号限流（`AI_LAB_RATE_LIMIT_PER_MINUTE` / `AI_LAB_RATE_LIMIT_PER_DAY`）。设置 `AI_ENABLED=false` 可显式关闭。详见 `.env.example` 中 AI 配置段。

**原则：** 新用户先跑通核心；需要业务扩展再开进阶，并准备对应运维。

### 模块开关（二次开发推荐）

| 变量 | 默认 | 说明 |
|------|------|------|
| `MODULE_ORDERS_ENABLED` | `true` | 关闭后隐藏订单菜单并拒绝订单 API |
| `MODULE_PAYMENTS_ENABLED` | `false` | 默认关闭（支付网关能力未成品）；`true` 仅用于管理端演示 UI |
| `APP_ENABLE_DEV_TOOL` | `false` | 生产显式 `true` 才开放开发工具。表单演示：`local/development/test` 默认可见；代码生成器：仅 `local/development` 默认可见（`test` 默认隐藏） |

登录 `Info` 与 `menus/tree` 会按开关过滤菜单；前端 `userStore.config` 同步 `orders_enabled` 等字段。

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

# 建议：限制管理端域名（可按需）
# DOMAINS_ADMIN=admin.example.com

# 生产默认关闭
SWAGGER_ENABLED=false
# APP_DISABLED_RUNNERS=  # 不要误关 queue:*
```

**上线检查（最小）：**

1. `migrate` 成功，`db:seed` 后修改默认 `admin` 密码  
2. Redis 可用，Web 进程与 Queue Worker 常驻  
3. HTTPS + 反向代理  
4. 关闭或限权：Swagger、pprof、代码生成器  
5. 日志磁盘与备份策略就绪  
6. 使用 `.env.production.example` 起步，**勿**直接用 `docker-compose.yml` 默认口令上生产  

部署细节见 [PRODUCTION.md](./PRODUCTION.md)（健康检查 `/health` `/ready`、告警与上线清单）、[BUILD.md](./BUILD.md)、[DOCKER_DEPLOY.md](./DOCKER_DEPLOY.md)。

### 资源归属（管理端）

- **导出**：下载 / 进度 SSE / 删除仅本人或配置的 `admin.super_admin_id`  
- **附件**：私有文件读/写同归属规则；公开附件（`is_public=1`）已登录管理员可读，改删仍需所有者或超管  
- **支付**：查询 / 回调能力返回 `payment_gateway_not_implemented`（501），勿当收单核心

---

## 4. 完整进阶配置（可选）

在「最小生产」之上，按模块叠加：

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

详见 [SEARCH.md](./SEARCH.md)。

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

需要搜索集群 + `queue-search` Worker。Outbox 积压可用 `go run . artisan search:retry-outbox` 补偿。初始化：`search:init-orders-index`，全量：`search:sync-orders`。其他业务索引用 `search.RegisterDefinition` 扩展（见 [SEARCH.md](./SEARCH.md)）。

### 4.4 OpenTelemetry

```ini
# OTEL_TRACES_EXPORTER=otlptrace
# OTEL_METRICS_EXPORTER=otlpmetric
# OTEL_EXPORTER_OTLP_TRACES_ENDPOINT=http://127.0.0.1:4318
```

未配置 exporter 时框架会自动禁用 `goravel:telemetry` runner，减少噪音。

---

## 5. 开源发布检查清单

```text
□ README 写清适用 / 不适用场景
□ .env.example / .env.production.example 可对照最小生产配置
□ migrate + seed 可一键初始化
□ CI：unit + feature（MySQL/Redis）+ Vue/React type-check/build
□ 演示账号与生产密钥分离说明
□ 进阶模块（分表 / ES / 队列）标注为可选
□ 冒烟集成测试：登录、鉴权接口、基础业务读接口
□ 导出 / 私有附件具备归属校验
```

---

## 6. 相关文档

| 文档 | 说明 |
|------|------|
| [QUICKSTART_DOCKER.md](./QUICKSTART_DOCKER.md) | Docker 本地三分钟跑通 |
| [BUILD.md](./BUILD.md) | 编译与部署 |
| [TESTING.md](./TESTING.md) | 测试指南 |
| [SHARDING_MIGRATION.md](./SHARDING_MIGRATION.md) | 分表 |
| [ERROR_CODES.md](./ERROR_CODES.md) | 错误码 |
| [ARCHITECTURE.md](./ARCHITECTURE.md) | 架构 |
| [TENANT_RESERVED.md](./TENANT_RESERVED.md) | 一户一库 / PG Schema 多租户（`TENANCY_DRIVER=database`） |
| [SEARCH.md](./SEARCH.md) | Elasticsearch / Meilisearch 切换、订单同步、文章扩展 |
| [PRODUCTION.md](./PRODUCTION.md) | 生产上线清单、`/health` `/ready`、告警 |
