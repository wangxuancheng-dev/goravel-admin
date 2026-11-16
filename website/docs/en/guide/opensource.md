# Open-source scope & modules

Chinese full version: [开源定位与模块](/guide/opensource) (authoritative for tables and Chinese wording).

## Suggested reading path (narrow first)

1. [Quick start](/en/guide/getting-started) — run locally (single DB by default)
2. This page — module boundaries and **capability tiers** (do not turn everything on at once)
3. [Architecture](/en/guide/architecture) → [Development](/en/guide/development) / [Code generator](/en/guide/code-generator)
4. Go-live: [Production](/en/deploy/production) ([minimal tenancy path](/en/deploy/production#minimal-tenancy-path)) → [SaaS checklist](/en/advanced/saas)
5. Tenant business code: [Tenancy developer guide](/en/advanced/tenancy-dev); many tenants: [Scale](/en/advanced/tenancy-scaling)

**Frontend:** React (`html-react/`) is primary; Vue (`html/`) follows for parity — [Frontend parity](/en/guide/frontend-parity).

## Fit / not a fit

**Good fit**

- Mid-size multi-tenant SaaS ops (DB-per-tenant, platform console, provision/migrate gate)
- Internal admin panels, ops backends, management mid-tier
- Goravel + Vue / React starter
- RBAC, menus, logs, export, code generator
- Small/medium business modules (users, orders, payments as optional demos)

**Not a fit**

- Financial trading cores or strong-consistency payment hubs (payments here are admin + gateway sample)
- Huge multi-region commercial SaaS platforms with hard SLAs
- Turning on sharding + ES without ops planning

Demo accounts are for exploration only. Change default admin password and secrets for production.

## Payment boundary

| Capability | Status |
|------------|--------|
| Payment method CRUD, payment records list/detail/export | Available |
| Create payment for order + optional `initiate` | Reference flow |
| **Mock** gateway pay / query / notify → `ApplyPaidResult` | Runnable locally |
| WeChat / Alipay client calls (gopay) | Sample; needs your merchant config |
| WeChat / Alipay query & notify verify | `payment_gateway_not_implemented` (501) |
| New channels | `app/payment/gateways` + `RegisterGateway` + `notify/{type}` |
| Refund / original-path refund | Not provided |

On public deploys, restrict gateways with `PAYMENT_GATEWAYS_ENABLED` and withhold payment permissions from roles that should not use them.

## Capability tiers (keep features; expose in layers)

Do not delete modules — stage what you must learn first. Full Chinese tables: [开源定位 §2](/guide/opensource#2-能力分档不删功能分层暴露).

| Tier | When | Examples |
|------|------|----------|
| **A — default on** | Public / DB-per-tenant go-live | JWT/RBAC, Redis cache+queue, [minimal tenancy path](/en/deploy/production#minimal-tenancy-path), API/Worker split |
| **B — by scale** | More tenants / noisier load | Pool caps, fleet concurrency, backup disk/schedule — [Scale](/en/advanced/tenancy-scaling). Mid-size (&lt;~50 active) usually skips these at first |
| **C — by product** | Optional / demo | Search, sharding, entitlement enforcement, vanity domains, real payment verify, AI Lab, OTel |

**Module switches**

| Variable | Default | Effect |
|----------|---------|--------|
| `PAYMENT_GATEWAYS_ENABLED` | empty | Gateway allowlist (e.g. `wechat,alipay`); empty / `*` / `all` = all registered. **Prefer explicit production allowlist**; new channels: [Payments](/en/advanced/payments) §6 |
| `APP_ENABLE_DEV_TOOL` | `false` | Explicit `true` in production to open dev tools |

## Data scope (row-level, inside a tenant)

Configured on **roles** (`roles.data_scope`); widest wins across roles; `super-admin` always sees all. Orthogonal to DB-per-tenant isolation.

| Value | Meaning |
|-------|---------|
| 1 | All data |
| 2 | Custom departments (`role_department`) |
| 3 | Own department |
| 4 | Department and children |
| 5 | Self only |

Wired lists: admins (`department_id`), articles / attachments / export jobs (`admin_id`).

## Field desensitization (list / export)

Masks configured PII on **list** and **export** responses. **Detail/show stays plain** so edit forms do not save masked values. Roles in bypass slugs (default `super-admin`) see plaintext.

**Per tenant / site:** Admin → Config → Desensitize writes `configs` group `desensitize`. With `TENANCY_DRIVER=database`, each tenant DB is independent — only tenants that need masking turn it on. Missing DB rows fall back to env.

| Env / config | Default | Notes |
|--------------|---------|-------|
| `DESENSITIZE_ENABLED` | `true` | Process default (`config/desensitize.go`) |
| `configs.desensitize.enabled` | (env) | Per-tenant / single-site override |
| `DESENSITIZE_BYPASS_ROLE_SLUGS` / DB `bypass_role_slugs` | `super-admin` | Comma-separated role slugs |
| modules | `user` / `admin` → `phone`, `email` | Strategies: `phone`, `email`, `hide`, `keep_ends` |

Wired: user list + user CSV export; admin list + admin sync export. Update ignores values that already look masked (`*`).

## Minimal production sketch

```ini
APP_ENV=production
APP_DEBUG=false
APP_KEY=          # go run . artisan key:generate
JWT_SECRET=       # strong random
CACHE_STORE=redis
QUEUE_CONNECTION=redis
SWAGGER_ENABLED=false
```

See [Production checklist](/en/deploy/production) (health `/health` `/ready`, Task Center / import permissions, queue backlog alert), [Build](/en/deploy/build), [Docker deploy](/en/deploy/docker). After upgrades that ship Task Center, re-check menu `export/TaskCenter` and `import.*` slugs per that checklist.
