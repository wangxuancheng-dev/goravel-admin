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

# Production binary
./main artisan db:seed --seeder=MenuSeeder
./main artisan db:seed --seeder=PermissionSeeder
```

`PermissionSeeder` upserts import permissions under the export menu; `MenuSeeder` keeps `Component: export/TaskCenter`. Full `db:seed` is for first-time init only. Non-super roles must be granted the new slugs in Role management.

## Health

- `/health` — liveness  
- `/ready` — readiness (DB / critical deps)

Do **not** put queue backlog scans on `/ready` (latency). Use the scheduled command below.

### Queue backlog alert (offline)

`queue:alert-backlog` (hourly in `app/console/kernel.go`) sums Redis pending via `QueueStatsReader`; POSTs webhook when over threshold (1h cache debounce). Webhook URL: `QUEUE_ALERT_WEBHOOK_URL`, else `READY_ALERT_WEBHOOK_URL`. Threshold: `QUEUE_ALERT_BACKLOG_THRESHOLD` (default `100`).

```bash
./main artisan queue:alert-backlog
```

Ready failure webhook: `READY_ALERT_WEBHOOK_URL` (5 min debounce on `/ready` non-200).

## Resource ownership (admin)

- **Exports:** download / SSE progress / delete — owner or configured `admin.super_admin_id`  
- **Attachments:** private R/W same rule; public (`is_public=1`) readable by logged-in admins; mutate still owner/super  
- **Payments:** use **mock** for end-to-end; WeChat/Alipay query & notify verify still stub (`payment_gateway_not_implemented`)

## Admin SPA (React) & Docker

**React (`html-react/`) is the primary shipping UI**; Vue (`html/`) is the peer reference. Images default to `BUILD_FRONTEND=0` (API-only). Set `BUILD_FRONTEND=1` to build **React** into `public/admin`.

```bash
cd html-react && npm ci && npm run build
docker build --build-arg BUILD_FRONTEND=1 -t goravel-admin .
```

Full Chinese detail: [生产清单](/deploy/production) §6.

## Related

- [Build & deploy](/en/deploy/build)  
- [Docker production](/en/deploy/docker)  
- [Testing](/en/guide/testing)  
- [Open-source scope](/en/guide/opensource)  

