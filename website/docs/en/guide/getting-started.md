# Docker quick start

For new users: bring up **MySQL + Redis + API** with Docker Compose in a few commands.

For production blue/green style deploy, see [Docker production](/en/deploy/docker).

## Prerequisites

- Docker / Docker Compose v2
- Free local ports `3000` / `3306` / `6379` (remap in compose if needed)

## Three steps

```bash
cp .env.docker.example .env
docker compose up -d --build
# Open http://localhost:3000
# Default admin / admin123 — change the password
```

Defaults:

| Item | Behavior |
|------|----------|
| Migrate | `artisan migrate` on container start |
| Seed | `db:seed` when `RUN_SEED=true` (seed failure does not block startup) |
| Queue | `QUEUE_CONNECTION=sync` in compose (single container is enough) |
| Env file | Root `.env` required (copy from `.env.docker.example`) |

Disable auto-seed with `RUN_SEED=false` in `.env`.

> Default is **single database** (`TENANCY_DRIVER=off`). To enable database-per-tenant with Docker locally, see [Multi-tenancy · Docker local](/en/advanced/tenancy#docker-local-setup).

## Frontend development

Compose starts the API only. Frontends:

```bash
# Vue
cd html && cp .env.example .env && npm i && npm run dev   # :3007

# React
cd html-react && cp .env.example .env && npm i && npm run dev  # :3008
```

`.env.docker.example` already allows CORS for those Vite origins.

## Useful commands

```bash
docker compose logs -f app
docker compose exec app /www/main artisan db:seed
docker compose down          # keep volumes
docker compose down -v       # wipe MySQL/Redis data
```
