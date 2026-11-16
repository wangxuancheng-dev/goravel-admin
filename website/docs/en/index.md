---
layout: home

hero:
  name: Goravel Admin
  text: Mid-size SaaS ops + admin starter
  tagline: Goravel · DB-per-tenant platform · React (primary) / Vue · RBAC · Docker
  actions:
    - theme: brand
      text: Quick start
      link: /en/guide/getting-started
    - theme: alt
      text: Open-source scope
      link: /en/guide/opensource
    - theme: alt
      text: GitHub
      link: https://github.com/wangxuancheng-dev/goravel-admin

features:
  - title: Suggested reading order
    details: Quick start → open-source scope (capability tiers) → architecture → development → minimal production path / SaaS checklist. Scale knobs, search, sharding on demand.
  - title: Platform ops
    details: Database-per-tenant, platform console, provision → migrate → ready gate, health/ready and queue alerts.
  - title: Dual frontend
    details: React (html-react) is the primary UI; Vue stays API-parity. See frontend parity.
  - title: Core and advanced
    details: RBAC, logs, export, codegen out of the box. Search, sharding, entitlements, vanity domains are tier C opt-ins.
---

## Docs navigation

| Section | Notes |
|---------|-------|
| [Quick start](/en/guide/getting-started) | Local or Docker first run |
| [Open-source scope](/en/guide/opensource) | Boundaries, **capability tiers**, reading path |
| [Architecture](/en/guide/architecture) | Before secondary development |
| [Development](/en/guide/development) / [Code generator](/en/guide/code-generator) | Adding CRUD (generator-first) |
| [Production](/en/deploy/production) / [SaaS checklist](/en/advanced/saas) | Before go-live |
| [Tenancy developer guide](/en/advanced/tenancy-dev) | Short page before writing tenant business code |
| [Advanced](/en/advanced/tenancy) / [Scale](/en/advanced/tenancy-scaling) / [Reference](/en/reference/api) | Full tenancy ops, scale knobs, payments/sharding |

Tenant demo: https://acme.xuancheng888.top/login?tenant_code=acme
Platform ops: https://admin.xuancheng888.top/platform/login
Credentials (both): demo / demo123 (not for production)
