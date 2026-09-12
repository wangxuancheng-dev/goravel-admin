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
| 队列堆积 / failed_jobs 增长 | Worker 存活、Redis、导出任务 |
| 磁盘（日志、`storage/backups`） | 备份与日志轮转 |
| MySQL `Threads_connected` 接近 `max_connections` | 下调 `TENANCY_POOL_*` 或扩容 |
| 证书到期 | HTTPS |

可选：配置 `OTEL_*` 接入 Jaeger/Grafana（见 OPENSOURCE 进阶段）。

## 4. 进程与备份

- Web：`go run .` / 编译产物常驻  
- Queue Worker：与 Web 分离，消费 `default` + 长任务队列（见 `bootstrap` runners）  
- 定时：`schedule:run` 或框架 schedule runner  
- 备份：平台库 + 各租户库（`tenant:backup` / `tenant:backup-all`）；公网务必异地副本，不要只留本机 `storage/backups`

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

## 6. 管理端 SPA（Vue）与 Docker

默认 **Vue（`html/`）是主发货 UI**；React（`html-react/`）为对等展示实现，不随镜像默认打包。

本地 / CI 单独构建前端：

```bash
cd html && npm ci && npm run build
# 产物在 html/dist；可用 nginx 反代，或拷到 public/admin 由 Go 静态托管
```

Docker 镜像支持可选同镜像内嵌 SPA：

```bash
# 仅 API（默认）：不构建前端，public/admin 可能为空目录
docker build -t goravel-admin .

# 构建 Vue 并复制到 public/admin
docker build --build-arg BUILD_FRONTEND=1 -t goravel-admin .
```

健康检查使用 **`GET /ready`**（就绪，含 DB/Redis），Dockerfile `HEALTHCHECK` 与 blue/green compose 已对齐；存活仍可用 `GET /health`。

通知渠道（邮件 / Webhook）见环境变量：`NOTIFICATION_MAIL_ENABLED`、`NOTIFICATION_WEBHOOK_ENABLED`、`NOTIFICATION_WEBHOOK_URL`。
