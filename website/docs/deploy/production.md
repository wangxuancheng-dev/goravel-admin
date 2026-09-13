# 生产上线清单与健康检查

面向公网部署的运维约定。功能边界见 [开源定位](/guide/opensource)；多租户见 [多租户](/advanced/tenancy)；上线核对见 [SaaS 核对清单](/advanced/saas)。

## 1. 启动前

1. 用 [`.env.production.example`](../.env.production.example) 生成 `.env`
2. `go run . artisan key:generate`，填写强随机 `JWT_SECRET`
3. `CACHE_STORE=redis`、`QUEUE_CONNECTION=redis`（勿用 `memory` / `sync` 上公网多实例）
4. `APP_DEBUG=false`，关闭 Swagger / 代码生成器 / pprof
5. `MODULE_PAYMENTS_ENABLED=false`
6. 一户一库时：`TENANCY_RESOLVER=subdomain`，`TENANCY_ALLOW_PLATFORM_DB_CREDENTIALS=false`
7. `migrate`（平台库）；租户库用平台 UI 异步迁移或 `tenant:migrate` / `tenant:migrate-all`；异步运维需 `long-running` worker
8. 修改默认管理员密码；平台管理员用 `platform:install`

应用在 `APP_ENV=production` 时会对不安全默认项打 **Warning** 日志（不阻断启动），见 `app/production/warn.go`。

## 1.1 任务中心与导入权限（升级后必查）

升级含「导入导出 / 任务中心」的版本后，确认菜单与权限已落库，否则侧边栏无入口或导入相关按钮 403。

| 项 | 说明 |
|----|------|
| 菜单 | 系统管理 →「导入导出」，组件路径必须为 `export/TaskCenter`（React / Vue 同路径约定） |
| 导入权限 slug | `import.index`、`import.show`、`import.download_error`、`import.destroy` |
| 模块导入按钮 | 生成器模块另有 `{module}.import`（如 `article.import`），需在角色中勾选 |

**推荐（增量、幂等）：** 重新跑权限 / 菜单 seeder，或只在角色 UI 里勾选上述 slug：

```bash
# 开发
go run . artisan db:seed --seeder=MenuSeeder
go run . artisan db:seed --seeder=PermissionSeeder

# 生产二进制
./main artisan db:seed --seeder=MenuSeeder
./main artisan db:seed --seeder=PermissionSeeder
```

`PermissionSeeder` 会按 slug 幂等写入导入相关权限并挂到「导入导出」菜单；`MenuSeeder` 确保 `Component: export/TaskCenter`。完整 `db:seed` 仅适合首次初始化。非超管角色需在 **角色管理** 中手动勾选新权限。

开源边界与上线总览见 [开源定位](/guide/opensource)。

## 2. 健康检查端点

| 路径 | 用途 | 成功 | 失败 |
|------|------|------|------|
| `GET /health` | **存活**（进程在） | 200 `status=healthy`，可选 `app`/`env`/`version` | 进程挂了才无响应 |
| `GET /ready` 或 `GET /health/ready` | **就绪**（可接流量） | 200 `{"status":"ready","checks":[...]}` | **503** `not_ready` |

就绪检查：

- **database**：默认库 `Ping`（始终）
- **redis**：当 `CACHE_STORE=redis` 或队列驱动为 redis/redisstream 时 Ping；否则 `skipped`
- **search**：当 `SEARCH_ENABLED=true` 时对当前驱动 `Ping`；否则 `skipped`

Kubernetes 示例：

```yaml
livenessProbe:
  httpGet: { path: /health, port: 3000 }
  initialDelaySeconds: 10
  periodSeconds: 10
readinessProbe:
  httpGet: { path: /ready, port: 3000 }
  initialDelaySeconds: 5
  periodSeconds: 5
```

本地轮询：`scripts/monitor_health.sh http://127.0.0.1:3000`

平台控制台另有鉴权后的 `GET /api/platform/health`（租户计数等），不用于 LB 探针。

## 3. 建议告警

| 信号 | 建议 |
|------|------|
| `/ready` 连续 503 | 页面告警；查 DB/Redis |
| 5xx 比例升高 | 网关/日志告警 |
| 队列堆积 / failed_jobs 增长 | Worker 存活、Redis、导出/导入任务；可选 `queue:alert-backlog`（见下） |
| 磁盘（日志、`storage/backups`） | 备份与日志轮转 |
| MySQL `Threads_connected` 接近 `max_connections` | 下调 `TENANCY_POOL_*` 或扩容 |
| 证书到期 | HTTPS |

可选：配置 `OTEL_*` 接入 Jaeger/Grafana（见 OPENSOURCE 进阶段）。

## 4. 进程与备份

- Web：`go run .` / 编译产物常驻  
- Queue Worker：与 Web 分离，消费 `default` + `long-running`（及可选 `search`）队列（见 `bootstrap/runners.go`）  
- 定时：`schedule:run` 或框架 `goravel:schedule` runner（多机时只留一台）  
- 备份：平台库 + 各租户库（`tenant:backup` / `tenant:backup-all`）；公网务必异地副本，不要只留本机 `storage/backups`

### 4.1 多机：API 与 Queue Worker 分角色

同一二进制、同一套云 Redis / 云库；按角色改 `.env`，不要每台都跑 HTTP + 全量队列。

```
用户 → LB → API ×N（只接 HTTP）
              ↓ Dispatch
           Redis 队列
              ↓
           Worker ×M（只消费）
```

| 角色 | 做什么 | `APP_DISABLED_RUNNERS` | 说明 |
|------|--------|--------------------------|------|
| **API** | 对外 HTTP，入队 | `queue-*` | 关闭 `queue-default` / `queue-long-running` / `queue-search`，避免与 Worker 抢任务、占连接 |
| **Worker** | 消费队列 | 留空（勿禁用 `queue-*`） | `QUEUE_CONNECTION=redis`；导入/导出/租户运维依赖 `long-running` |
| **定时（可选）** | 只跑 schedule | API 副本可加 `goravel:schedule` | 多机时 schedule **只开一台**，防止重复执行 |

**API 机示例：**

```ini
CACHE_STORE=redis
QUEUE_CONNECTION=redis
APP_DISABLED_RUNNERS=queue-*
# 若本机也不跑定时：APP_DISABLED_RUNNERS=queue-*,goravel:schedule
```

**Worker 机示例：**

```ini
CACHE_STORE=redis
QUEUE_CONNECTION=redis
# 不要设置 APP_DISABLED_RUNNERS=queue-*
QUEUE_CONCURRENT=2
QUEUE_LONG_RUNNING_CONCURRENT=1
```

两端都启动同一编译产物（如 `./main`）。`QUEUE_CONNECTION=sync` 时任务不入 Redis，多机无法分担，生产勿用。

Runner 名是连字符 `queue-*`（见 `bootstrap/runners.go`），不是 `queue:*`。`queue:*` 用于生产 Artisan 命令白名单过滤，不能用来关队列 Runner。

## 5. 上线最短路径

```bash
cp .env.production.example .env
# 填 APP_KEY / JWT / DB / Redis / APP_URL / CORS
go run . artisan migrate
# 若 TENANCY_DRIVER=database：
# go run . artisan platform:install -u ... -p ...
# go run . artisan tenant:create acme "Acme" --migrate   # 或带独立 DB 账号
curl -sf http://127.0.0.1:3000/health
curl -sf http://127.0.0.1:3000/ready
```

## 6. 管理端 SPA（React）与 Docker

默认 **React（`html-react/`）是推荐前端**；Vue（`html/`）为对等实现。镜像默认不构建前端（`BUILD_FRONTEND=0`，加快纯 API 镜像）；需要同镜像托管 SPA 时设 `BUILD_FRONTEND=1`，**优先构建 `html-react/`** 并复制到 `public/admin`。

本地 / CI 单独构建前端：

```bash
cd html-react && npm ci && npm run build
# 产物在 html-react/dist；可用 nginx 反代，或拷到 public/admin 由 Go 静态托管
# Vue 参考端：cd html && npm ci && npm run build
```

Docker 镜像支持可选同镜像内嵌 SPA：

```bash
# 仅 API（默认）：不构建前端，public/admin 可能为空目录
docker build -t goravel-admin .

# 构建 React（html-react）并复制到 public/admin
docker build --build-arg BUILD_FRONTEND=1 -t goravel-admin .
```

健康检查使用 **`GET /ready`**（就绪，含 DB/Redis），Dockerfile `HEALTHCHECK` 与 blue/green compose 已对齐；存活仍可用 `GET /health`。

通知渠道（邮件 / Webhook）见环境变量：`NOTIFICATION_MAIL_ENABLED`、`NOTIFICATION_WEBHOOK_ENABLED`、`NOTIFICATION_WEBHOOK_URL`；类型白名单 `NOTIFICATION_MAIL_TYPES` / `NOTIFICATION_WEBHOOK_TYPES`（逗号分隔，空=全部）。就绪失败告警：`READY_ALERT_WEBHOOK_URL`（`/ready` 非 200 时 POST JSON，缓存防抖 5 分钟）。

### 队列积压告警（离线，不进 `/ready`）

定时命令 `queue:alert-backlog`（默认每小时，见 `app/console/kernel.go`）在默认队列连接为 Redis 时汇总 pending；超过阈值则 POST Webhook（缓存防抖 1 小时）。**不**挂在 `/ready` 请求路径上，避免拉高探针延迟。

| 变量 | 说明 |
|------|------|
| `QUEUE_ALERT_WEBHOOK_URL` | 积压告警 URL；空则回退 `READY_ALERT_WEBHOOK_URL`；皆空则关闭 |
| `QUEUE_ALERT_BACKLOG_THRESHOLD` | pending 合计阈值，默认 `100` |

```bash
./main artisan queue:alert-backlog
# 或手动：go run . artisan queue:alert-backlog
```
