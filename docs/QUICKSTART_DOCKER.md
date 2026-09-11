# Docker 本地一键启动

面向新用户：用 Docker Compose 拉起 **MySQL + Redis + API**，约三条命令即可登录后台。

生产蓝绿部署请看 [DOCKER_DEPLOY.md](./DOCKER_DEPLOY.md)。

## 前置

- Docker / Docker Compose v2
- 本机 `3000` / `3306` / `6379` 端口空闲（可在 compose 中改映射）

## 三步

```bash
cp .env.docker.example .env
docker compose up -d --build
# 浏览器打开 http://localhost:3000
# 账号 admin / admin123（务必改密）
```

默认行为：

| 项 | 说明 |
|----|------|
| 迁移 | 容器启动时 `artisan migrate` |
| 填充 | `RUN_SEED=true` 时执行 `db:seed`（失败不阻断启动，便于重复 up） |
| 队列 | compose 内默认 `QUEUE_CONNECTION=sync`，单容器即可 |
| 环境文件 | 必须有根目录 `.env`（由 `.env.docker.example` 复制） |

关闭自动填充：在 `.env` 设 `RUN_SEED=false`。

## 前端开发

Compose 只起后端。前端：

```bash
# Vue
cd html && cp .env.example .env && npm i && npm run dev   # :3007

# React
cd html-react && cp .env.example .env && npm i && npm run dev  # :3008
```

`.env.docker.example` 已包含上述 Vite 源的 CORS 放行。

## 常用命令

```bash
docker compose logs -f app
docker compose exec app /www/main artisan db:seed
docker compose down          # 保留数据卷
docker compose down -v       # 删除 MySQL/Redis 数据
```
