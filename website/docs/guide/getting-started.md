# 快速开始

两条路径任选：**本机开发**（自备 MySQL / Redis）或 **Docker Compose**（一键拉起依赖 + API）。二次开发一般用本机；只想先看效果可用 Docker。

生产蓝绿部署见 [Docker 生产部署](/deploy/docker)。

默认登录：`admin` / `admin123`（务必尽快改密）。

---

## 路径 A：本机开发（推荐日常） {#local-dev}

### 前置

| 工具 | 版本 |
|------|------|
| Go | 1.21+ |
| Node.js | 20+ |
| MySQL | 8.0+ |
| Redis | 7.0+（也可用下方「无 Redis」简化项） |

### 后端

```bash
git clone https://github.com/wangxuancheng-dev/goravel-admin.git
cd goravel-admin
go mod tidy

cp .env.example .env
# 编辑 .env：至少改 DB_*（库名 / 账号 / 密码），确认 APP_PORT=3000
```

可选简化（无 Redis 时）：

```env
CACHE_STORE=memory
QUEUE_CONNECTION=sync
```

有 Redis 时保持 `.env.example` 默认即可。

```bash
go run . artisan migrate
go run . artisan db:seed

go run . --no-ansi
# 热重载：air
```

API：`http://127.0.0.1:3000`。Swagger：`http://127.0.0.1:3000/swagger/index.html`。

### 前端（二选一）

```bash
# Vue → http://localhost:3007
cd html
cp .env.example .env   # 若尚无
npm i && npm run dev

# 或 React → http://localhost:3008
cd html-react
cp .env.example .env
npm i && npm run dev
```

`.env.example` 已放行 `3007` / `3008` 的 CORS。改 CORS 后需重启 Go 进程。多租户下子域与已激活独立域名的跨域见 [多租户](/advanced/tenancy)。

> 默认**单库**（`TENANCY_DRIVER=off`）。多租户见 [多租户](/advanced/tenancy)。

下一步：加业务 CRUD 看 [开发指南](/guide/development) / [代码生成器](/guide/code-generator)。

---

## 路径 B：Docker 一键跑通 {#docker}

适合不想本机装 MySQL/Redis，或先验证镜像能起。

### 前置

- Docker / Docker Compose v2
- 本机 `3000` / `3306` / `6379` 端口空闲（可在 compose 中改映射）

### 三步

```bash
cp .env.docker.example .env
docker compose up -d --build
# 浏览器打开 http://localhost:3000
```

| 项 | 说明 |
|----|------|
| 迁移 | 容器启动时 `artisan migrate` |
| 填充 | `RUN_SEED=true` 时执行 `db:seed`（失败不阻断启动） |
| 队列 | compose 内默认 `QUEUE_CONNECTION=sync` |
| 环境文件 | 根目录 `.env`（由 `.env.docker.example` 复制） |

关闭自动填充：`.env` 设 `RUN_SEED=false`。

> Docker 下一户一库多租户：见 [多租户 · Docker 本地开启](/advanced/tenancy#docker-本地开启)。

### 前端仍走本机 Vite

Compose 只起后端。前端同「路径 A」：

```bash
cd html && cp .env.example .env && npm i && npm run dev          # :3007
# 或
cd html-react && cp .env.example .env && npm i && npm run dev    # :3008
```

### 常用命令

```bash
docker compose logs -f app
docker compose exec app /www/main artisan db:seed
docker compose down          # 保留数据卷
docker compose down -v       # 删除 MySQL/Redis 数据
```
