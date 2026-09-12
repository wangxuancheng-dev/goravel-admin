# Quick start

Two options: **local development** (your own MySQL / Redis) or **Docker Compose** (dependencies + API in one shot). Day-to-day work usually uses local; Docker is fine for a first smoke test.

Production blue/green: [Docker production](/en/deploy/docker).

Default login: `admin` / `admin123` (change ASAP).

---

## Path A: Local development (recommended) {#local-dev}

### Prerequisites

| Tool | Version |
|------|---------|
| Go | 1.21+ |
| Node.js | 20+ |
| MySQL | 8.0+ |
| Redis | 7.0+ (or use the no-Redis tip below) |

### Backend

```bash
git clone https://github.com/wangxuancheng-dev/goravel-admin.git
cd goravel-admin
go mod tidy

cp .env.example .env
# Edit .env: set DB_* at least; keep APP_PORT=3000
```

Optional (no Redis):

```env
CACHE_STORE=memory
QUEUE_CONNECTION=sync
```

Otherwise keep the Redis defaults from `.env.example`.

```bash
go run . artisan migrate
go run . artisan db:seed

go run . --no-ansi
# hot reload: air
```

API: `http://127.0.0.1:3000`. Swagger: `http://127.0.0.1:3000/swagger/index.html`.

### Frontend (pick one)

```bash
# Vue → http://localhost:3007
cd html
cp .env.example .env   # if missing
npm i && npm run dev

# or React → http://localhost:3008
cd html-react
cp .env.example .env
npm i && npm run dev
```

`.env.example` already allows CORS for ports `3007` / `3008`. Restart Go after changing CORS.

> Default is **single database** (`TENANCY_DRIVER=off`). Multi-tenancy: [Tenancy](/en/advanced/tenancy).

Next: [Development guide](/en/guide/development) / [Code generator](/en/guide/code-generator).

---

## Path B: Docker quick start {#docker}

When you do not want to install MySQL/Redis locally, or only need a smoke test.

### Prerequisites

- Docker / Docker Compose v2
- Free ports `3000` / `3306` / `6379` (remap in compose if needed)

### Three steps

```bash
cp .env.docker.example .env
docker compose up -d --build
# Open http://localhost:3000
```

| Item | Behavior |
|------|----------|
| Migrate | `artisan migrate` on container start |
| Seed | `db:seed` when `RUN_SEED=true` (failure does not block startup) |
| Queue | `QUEUE_CONNECTION=sync` in compose |
| Env file | Root `.env` from `.env.docker.example` |

Disable auto-seed with `RUN_SEED=false`.

> Database-per-tenant under Docker: [Multi-tenancy · Docker local](/en/advanced/tenancy#docker-local-setup).

### Frontend still uses local Vite

Compose starts the API only. Same as Path A:

```bash
cd html && cp .env.example .env && npm i && npm run dev          # :3007
# or
cd html-react && cp .env.example .env && npm i && npm run dev    # :3008
```

### Useful commands

```bash
docker compose logs -f app
docker compose exec app /www/main artisan db:seed
docker compose down          # keep volumes
docker compose down -v       # wipe MySQL/Redis data
```
