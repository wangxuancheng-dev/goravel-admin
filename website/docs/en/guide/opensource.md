# Open-source scope & modules

Chinese full version: [开源定位与模块](/guide/opensource).

## Fit / not a fit

**Good fit**

- Internal admin panels, ops backends, management mid-tier
- Goravel + Vue / React starter
- RBAC, menus, logs, export, code generator
- Small/medium business modules (users, orders, payments as optional demos)

**Not a fit**

- Financial trading cores or strong-consistency payment hubs (payments here are admin + gateway sample)
- Huge multi-region commercial SaaS platforms with hard SLAs
- Turning on sharding + ES + multi-queue without ops planning

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

Enabled by default; set `MODULE_PAYMENTS_ENABLED=false` on public production unless you own the gateway (or mock-only).

## Core vs advanced

**Core:** JWT + RBAC, system management, operation/login/system logs, list export, code generator (dev). Minimal dependency: MySQL-compatible DB + Go process.

**Advanced (opt-in):** Redis cache/queue, order/payment sharding, Elasticsearch/Meilisearch, multi-queue drivers, OpenTelemetry, AI / pprof / Swagger, database-per-tenant (`TENANCY_DRIVER=database`).

**Module switches**

| Variable | Default | Effect |
|----------|---------|--------|
| `MODULE_ORDERS_ENABLED` | `true` | Hide order menus + reject order APIs when false |
| `MODULE_PAYMENTS_ENABLED` | `true` | Payment admin UI/API; set false on public deploy unless you own the gateway |
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
