# Production checklist

Ops conventions for public deploy. Feature boundaries: [Open-source scope](/en/guide/opensource). Multi-tenancy: [Tenancy](/en/advanced/tenancy). Go-live checklist: [SaaS checklist](/en/advanced/saas).

Full Chinese detail: [生产清单](/deploy/production).

## Before go-live

1. `migrate` succeeds; after `db:seed`, change default `admin` password  
2. Redis up; web process + queue workers stay running  
3. HTTPS + reverse proxy  
4. Disable or lock down: Swagger, pprof, code generator  
5. Log disk + backup policy ready  
6. Start from `.env.production.example` — **do not** ship `docker-compose.yml` default passwords to production  
7. Prefer `MODULE_PAYMENTS_ENABLED=false` unless you own the gateway  
8. After upgrades that add Task Center / CSV import: re-seed menus/permissions (next section)

## Task Center + import permissions

After shipping versions with import/export Task Center, confirm menu + permissions exist (otherwise no sidebar entry or 403 on import APIs).

| Item | Detail |
|------|--------|
| Menu | System → Import/Export; component path **`export/TaskCenter`** (React and Vue) |
| Import slugs | `import.index`, `import.show`, `import.download_error`, `import.destroy` |
| Module import | Generator modules also need `{module}.import` (e.g. `article.import`) on roles |

```bash
# Dev
go run . artisan db:seed --seeder=MenuSeeder
go run . artisan db:seed --seeder=PermissionSeeder

# Production binary (APP_ENV=production requires --force)
./main artisan db:seed --force --seeder=MenuSeeder
./main artisan db:seed --force --seeder=PermissionSeeder
```

`PermissionSeeder` upserts import permissions under the export menu; `MenuSeeder` keeps `Component: export/TaskCenter`. Full `db:seed` is for first-time init only. Non-super roles must be granted the new slugs in Role management. When `APP_ENV=production`, Goravel requires `--force` on `db:seed`.

## Health

- `/health` — liveness  
- `/ready` — readiness (DB / critical deps)

Do **not** put queue backlog scans on `/ready` (latency). Use the scheduled command below.

### Queue backlog alert (offline)

`queue:alert-backlog` (hourly in `app/console/kernel.go`) sums Redis pending via `QueueStatsReader`; POSTs webhook when over threshold (1h cache debounce). Webhook URL: `QUEUE_ALERT_WEBHOOK_URL`, else `READY_ALERT_WEBHOOK_URL`. Threshold: `QUEUE_ALERT_BACKLOG_THRESHOLD` (default `100`).

```bash
./main artisan queue:alert-backlog
```

In **Observability → Queue**, check whether the webhook is configured and use **Send test alert** (`observability.queue_alert_test`). Configure the URL via `.env` (not the UI).

White-label: System Config → Website (name / logo). Login uses `GET /api/admin/login/branding`.

Platform tenant list: filter by provision status, **Retry all failed migrates**, summary at `GET /api/platform/tenants/ops-summary`.

Ready failure webhook: `READY_ALERT_WEBHOOK_URL` (5 min debounce on `/ready` non-200).
## Processes: API vs Queue Worker

Same binary and shared Redis/DB. Split roles via `.env` — do not run full HTTP + all queue runners on every node.

Sizing by active-tenant band (API / Worker / DB hosts): [Tenancy · Scale](/en/advanced/tenancy#scale-and-recommended-settings).

| Role | Purpose | `APP_DISABLED_RUNNERS` |
|------|---------|--------------------------|
| **API** | HTTP only; dispatch jobs | `queue-*` (disables all `queue-*` runners) |
| **Worker** | Consume queues | leave empty (do **not** disable `queue-*`); needs **`queue-schedule`** for flexible cron |
| **Schedule** | Optional | on API replicas add `goravel:schedule`; run schedule on **one** node only |

**API node:**

```ini
CACHE_STORE=redis
QUEUE_CONNECTION=redis
APP_DISABLED_RUNNERS=queue-*
# APP_DISABLED_RUNNERS=queue-*,goravel:schedule
```

**Worker node:**

```ini
CACHE_STORE=redis
QUEUE_CONNECTION=redis
QUEUE_CONCURRENT=2
QUEUE_LONG_RUNNING_CONCURRENT=1
QUEUE_SCHEDULE_CONCURRENT=10
```

Use `queue-*` (hyphen), not `queue:*`. The latter is for production Artisan command filters, not queue runners. See Chinese [生产清单](/deploy/production) §4.1.

## Resource ownership (admin)

- **Exports:** download / SSE progress / delete — owner or configured `admin.super_admin_id`  
- **Attachments:** private R/W same rule; public (`is_public=1`) readable by logged-in admins; mutate still owner/super  
- **Payments:** use **mock** for end-to-end; WeChat/Alipay query & notify verify still stub (`payment_gateway_not_implemented`)

## Admin SPA (React) & Docker

**React (`html-react/`) is the primary UI**; Vue (`html/`) is the peer implementation. Images default to `BUILD_FRONTEND=0` (API-only). Set `BUILD_FRONTEND=1` to build **React** into `public/admin`.

```bash
cd html-react && npm ci && npm run build
docker build --build-arg BUILD_FRONTEND=1 -t goravel-admin .
```

Full Chinese detail: [生产清单](/deploy/production) §6.

### Frontend static release: rollback & canary

The admin UI is a Vite static SPA (`html-react/dist` or `html/dist`). There is **no** built-in percentage canary switch; use the release method:

| Method | Rollback | Canary |
|--------|----------|--------|
| Nginx hosting `dist` | Keep previous release dir; flip symlink | Dual dirs / LB weighted split |
| Cloudflare Workers | Dashboard / wrangler previous version | Platform gradual deployments |
| Docker blue/green (`BUILD_FRONTEND=1`) | `scripts/deploy/rollback.sh` | **Full cutover** after health checks, not % traffic |

Server layout:

```text
/var/www/admin/
  current -> releases/20260914_1020
  releases/
    20260914_1000/
    20260914_1020/
```

Ship: unpack `dist` into `releases/<stamp>` → point `current` → `nginx -s reload`.  
Rollback: repoint `current` → reload. Keep the **whole** directory (hashed assets); do not replace only `index.html`.

Percentage canary needs Nginx/LB weights or Cloudflare Gradual Deployments. Repo blue/green scripts cover the **API container** (optional embedded SPA); see [Docker](/en/deploy/docker).

## Related

- Tenant scale / pool starting points: [Tenancy · Scale and recommended settings](/en/advanced/tenancy#scale-and-recommended-settings)
- [Build & deploy](/en/deploy/build)  
- [Docker production](/en/deploy/docker)  
- [Testing](/en/guide/testing)  
- [Open-source scope](/en/guide/opensource)  

