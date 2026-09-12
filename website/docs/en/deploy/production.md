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

## Health

- `/health` — liveness  
- `/ready` — readiness (DB / critical deps)

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

