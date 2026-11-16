# Multi-tenancy: scale and recommended settings

> Split out of [Multi-tenancy](/en/advanced/tenancy): production tuning and capacity planning. Business code only needs [Multi-tenancy: developer guide](/en/advanced/tenancy-dev).

These are **starting points**, not hard quotas — tune from monitoring. With many tenants the first bottleneck is usually the **per-tenant DB pool**, not Redis. Redis is shared cluster-wide (cache keys use `tenancy.CacheKey` → `t{id}:` prefix); size the instance up first. Split cache vs queue only if noisy-neighbor becomes real.

## What defaults can hold

"Active" means **recently trafficked tenants with a registered pool in-process**, not `COUNT(tenants)`.

| Scenario | Guidance |
|----------|----------|
| **Stock pool + single host** | Comfortable for about **&lt; 50 active**; stretch **50–200** (tighten `OPEN` first) |
| **Tuned pools + Redis queue + split API/Worker** | **Hundreds to ~1k active** |
| **Plus `tenants.host` DB sharding + `queue-schedule` workers** | **A few thousand active** (this product's target band) |
| **Plus regional multi-stack (cell) deployments** | **Beyond one fleet** — copy the stack (see below) |

Defaults: `TENANCY_POOL_MAX_OPEN_CONNS=20`, `TENANCY_REGISTERED_MAX=0` (unlimited). **Do not ship stock defaults unchanged for thousands of merchants.**

**Capacity formula (tenant DBs):**

`registered pools in process × TENANCY_POOL_MAX_OPEN_CONNS × API instances` ≪ DB `max_connections` (leave headroom for platform DB, backups, ops).

Multiple `tenant_*` databases on one MySQL/PG host still share that host's connection limit.

## Runtime dual-mode (small / large fleets)

| Mechanism | Config | Small fleet | Large fleet |
|-----------|--------|-------------|-------------|
| Idle registry eviction | `TENANCY_REGISTERED_IDLE_TTL` (default 900s) | Idle pools drop; reconnect on next request | Prevents registry growing to full fleet |
| Registry hard cap | `TENANCY_REGISTERED_MAX` (0=unlimited) | Optional | Prefer 300–500 |
| Fleet command paging | `TENANCY_SCOPE_AUTO_BATCH_AT` (200) / `TENANCY_SCOPE_AUTO_BATCH` (100) | ≤200 tenants: one pass | Above threshold: page; minute jobs rotate a cursor |
| Forced page size | `TENANCY_SCOPE_BATCH` | 0=auto | Force N per page |
| Flexible schedules | `next_run_at` + tick fan-out + `schedule` queue (`QUEUE_SCHEDULE_CONCURRENT`) | Same minute match semantics | Tick **enqueues only**; workers run handlers in parallel |
| Fleet backups | `tenant:backup-all` / `backup-scheduled` | Small fleets may enqueue all at once | Enqueue `long-running`; daily rotate `TENANT_BACKUP_SCHEDULE_BATCH` |
| Fleet migrate/seed | `TENANCY_MIGRATE_CONCURRENCY` / `--concurrency` | Default 2; same host prefer 2–5 | With many `tenants.host` shards, 20–50 OK (true parallel DDL/seed; do not stampede the DB) |

## Fleet schema migrate (`tenant:migrate-all`)

Database-per-tenant means one DDL/seed pass per tenant. Wall clock is roughly `tenants × per-tenant time / concurrency`. `MigrateTenant` / `SeedTenant` bind per-goroutine Schema+Orm (no global `database.default` flip), so `TENANCY_MIGRATE_CONCURRENCY` is **true parallelism** for both migrate-all and seed-all. Load still hits each MySQL/PG host that holds tenant DBs.

| Deploy | Suggested concurrency | Notes |
|--------|----------------------|-------|
| Single host / many DBs on one instance | 2–5 | Default 2; higher risks `max_connections` / disk IO |
| ~100 tenants per DB host, many hosts | 20–50 | Spread via `tenants.host` at provision; still a **global** cap, not per-host |
| Platform UI batch migrate/seed | `TENANCY_MIGRATE_CONCURRENCY` (`tenant_ops_fleet`) | Same path as CLI; `QUEUE_LONG_RUNNING_CONCURRENT=1` is enough |

Failed tenants get `provision_status=failed` + `last_migrate_error`; filter and retry without re-running the whole fleet. Cutover: new image platform `migrate`, then `tenant:migrate-all` (optional `--concurrency`), then switch traffic.

## Concurrency knobs (tune with queue workers)

These are **different** controls — raising one does not raise the others:

| Knob | What it limits | Typical use |
|------|----------------|-------------|
| `TENANCY_MIGRATE_CONCURRENCY` | **In-process** parallel tenants for `tenant:migrate-all` / `tenant:seed-all` (goroutines) | CLI fleet DDL/seed in a maintenance window |
| `QUEUE_LONG_RUNNING_CONCURRENT` | Jobs per long-running Worker | Per-tenant `tenant_ops`; **batch migrate/seed/backup only needs 1** (fleet uses one slot) |
| `QUEUE_SCHEDULE_CONCURRENT` | Jobs per schedule Worker | Minute collection / flexible schedules — **competes for tenant-DB IO** with migrate |
| `QUEUE_CONCURRENT` | Default queue parallelism | Exports, notifications, etc. |

**How to tune them together (same DB host):**

1. **During `migrate-all` / `seed-all`**: temporarily lower `QUEUE_SCHEDULE_CONCURRENT` to `1–2` (or pause the schedule tick), keep `QUEUE_LONG_RUNNING_CONCURRENT=1`, and **do not** also fire UI batch migrate — CLI + Worker concurrency would stack on the same MySQL/PG.
2. **Wall clock**: `≈ ceil(tenants / TENANCY_MIGRATE_CONCURRENCY) × per-tenant migrate time`; seed is usually much faster and can share the same concurrency value.
3. **Day-to-day**: `TENANCY_MIGRATE_CONCURRENCY` only affects CLI and may stay high; steady load is `QUEUE_SCHEDULE_CONCURRENT × Worker hosts` plus `TENANCY_POOL_MAX_OPEN_CONNS`.
4. **Rule of thumb**: same host prefer CLI concurrency `2–5`; use `20–50` only with many `tenants.host` shards. Keep long-running at `1`; scale schedule separately for collection SLA.
5. **Batch migrate/seed**: both CLI and platform UI batch read `TENANCY_MIGRATE_CONCURRENCY`. Do not raise `QUEUE_LONG_RUNNING_CONCURRENT` just for batch migrate. Still avoid running CLI and UI fleets against the same DB hosts at once.

**Platform UI batch migrate / seed / backup:**

1. **Batch** migrate/seed enqueues **one** `tenant_ops_fleet` job that uses `TENANCY_MIGRATE_CONCURRENCY` (same path as CLI `migrate-all`). Per-tenant ops still use `tenant_ops`. Keep `QUEUE_LONG_RUNNING_CONCURRENT=1` for batch migrate (the fleet occupies a single queue slot).
2. Worker hosts must run `queue-long-running` (do not set `APP_DISABLED_RUNNERS=queue-*`); API hosts usually Dispatch only.
3. After changing `.env`, **restart the Worker processes** that consume long-running (concurrency is read at startup in `bootstrap/runners.go`). Env-only change without Worker restart keeps the old concurrency. Restarting API alone does not change Worker parallelism.
4. Batch migrate speed comes from raising `TENANCY_MIGRATE_CONCURRENCY` (applies to newly enqueued fleets). Batch backup remains one job per tenant and still scales with `QUEUE_LONG_RUNNING_CONCURRENT`.

| Knob | Restart needed? | Notes |
|------|-----------------|-------|
| `QUEUE_LONG_RUNNING_CONCURRENT` / `QUEUE_SCHEDULE_CONCURRENT` / `QUEUE_CONCURRENT` | **Yes** — restart matching queue Workers | Concurrent is fixed when the runner starts |
| `TENANCY_MIGRATE_CONCURRENCY` | Next `artisan tenant:migrate-all` picks it up; no need to restart long-lived API/Workers for CLI | Or use one-shot `--concurrency=N` without editing env |
| Clicking migrate in the UI | Enqueue works without restart | Parallelism still follows already-running Worker concurrency |

**Suggested values (copy-paste, one tenant-DB host):**

| Scenario | `TENANCY_MIGRATE_CONCURRENCY` (CLI / UI batch migrate·seed) | `QUEUE_LONG_RUNNING_CONCURRENT` (per-tenant / batch backup) | `QUEUE_SCHEDULE_CONCURRENT` |
|----------|-------------------------------------|--------------------------------------|----------------------------|
| Local / small fleet &lt;50 | **2–3** (default 2) | **1** | **5–10** (often 10) |
| Same host, cutover `migrate-all` | **3–5** | **1** (do not raise during cutover) | Temporarily **1–2** or pause tick |
| ~100 tenants/DB host, sharded `tenants.host` | **20–40** | **1–2** | **5–10** |
| Few thousand, many hosts | **40–50** (max 100, use sparingly) | **1–2**; add Worker hosts for more speed | Start around **10**, tune to SLA |

Rule of thumb: UI batch migrate tunes `TENANCY_MIGRATE_CONCURRENCY` (same as CLI); keep `QUEUE_LONG_RUNNING_CONCURRENT=1`. Wall clock ≈ `ceil(tenants / concurrency) × per-tenant migrate seconds` (this repo: often ~10–20s DDL per tenant on a typical laptop/dev MySQL).

## Active-tenant bands

| Active tenants | Tenant pool | Queue / processes | Redis |
|----------------|-------------|-------------------|-------|
| &lt; 50 | Defaults `IDLE=2` / `OPEN=20` OK; can use `OPEN=10` | Worker may share API host; `QUEUE_CONNECTION=redis` | Single or small managed Redis |
| 50–200 | `IDLE=1–2`, `OPEN=3–5`; shorter idle/lifetime (e.g. 120 / 600) | Split **API vs Worker** (see [Production](/en/deploy/production)); keep `QUEUE_LONG_RUNNING_CONCURRENT` low | Managed Redis; watch `used_memory` and backlog |
| 200–1000 | `OPEN=2–3`; optional `REGISTERED_MAX=300` | Dedicated Workers: `default` + `long-running` + **`schedule`** | Larger tier; watch export/import neighbors |
| Few thousand | `OPEN=2`; `REGISTERED_MAX=300–500`; shard via `tenants.host`; migrate may use `TENANCY_MIGRATE_CONCURRENCY=20–50` | Scale Workers horizontally; tune `QUEUE_SCHEDULE_CONCURRENT` | HA Redis; split cache/queue if needed |

## Machine sizing and role layout (rule of thumb)

Same binary; split roles with `APP_DISABLED_RUNNERS` (see [Production](/en/deploy/production)). vCPU/RAM are cloud-host ballparks — adjust for QPS, collection duration, and export load.

| Active band | API | Queue Worker | Scheduler | Platform DB | Tenant DBs | Redis |
|-------------|-----|--------------|-----------|-------------|------------|-------|
| &lt; 50 | 1× 2vCPU 4GB (Worker may colocated) | Optional merge | Merged or 1 host | 1× small/mid | Same host multi-DB OK | 1× small |
| 50–200 | 2× 2vCPU 4GB (HTTP only) | 1× 2vCPU 4GB | **Exactly 1** schedule host | 1× (raise `max_connections`) | Same host or start 1 tenant DB host | 1× managed |
| 200–1000 | 2–4× API | 2+× Workers (include `schedule`) | 1 host | Dedicated platform DB | **Multiple** hosts via `tenants.host` | Mid/large HA |
| Few thousand | 4+× API behind LB | 3+× Workers; tune `QUEUE_SCHEDULE_CONCURRENT` | 1 host | Dedicated + backups | Many tenant DB hosts | HA; split if noisy |

**Roles:**

| Role | `APP_DISABLED_RUNNERS` | Queue runners needed |
|------|-------------------------|----------------------|
| API | `queue-*` | None (Dispatch only) |
| Worker | empty | `queue-default`, `queue-long-running`, **`queue-schedule`**, optional `queue-search` |
| Scheduler | may disable `queue-*` | Only `goravel:schedule`; **one host cluster-wide** |

Minute-collection throughput ≈ `QUEUE_SCHEDULE_CONCURRENT × Worker hosts`. Example: 3000 tenants/min at 5s each needs ~**250** concurrent slots (or lower frequency).

## Example configs

~100 active tenants, 2 API instances:

```ini
TENANCY_POOL_MAX_IDLE_CONNS=1
TENANCY_POOL_MAX_OPEN_CONNS=5
TENANCY_POOL_CONN_MAX_IDLETIME=120
TENANCY_POOL_CONN_MAX_LIFETIME=600
TENANCY_REGISTERED_IDLE_TTL=900

CACHE_STORE=redis
QUEUE_CONNECTION=redis
QUEUE_CONCURRENT=2
QUEUE_LONG_RUNNING_CONCURRENT=1
QUEUE_SCHEDULE_CONCURRENT=5
# API hosts: APP_DISABLED_RUNNERS=queue-*
```

Few-thousand active starting point:

```ini
TENANCY_POOL_MAX_IDLE_CONNS=1
TENANCY_POOL_MAX_OPEN_CONNS=2
TENANCY_REGISTERED_IDLE_TTL=900
TENANCY_REGISTERED_MAX=400
TENANCY_SCOPE_AUTO_BATCH_AT=200
TENANCY_SCOPE_AUTO_BATCH=100
TENANCY_FLEX_SCHEDULE_TICK_LIMIT=2000
TENANCY_FLEX_SCHEDULE_QUEUE=schedule
TENANCY_MIGRATE_CONCURRENCY=40
QUEUE_SCHEDULE_CONCURRENT=10
TENANT_BACKUP_SCHEDULE_ENABLED=true
TENANT_BACKUP_SCHEDULE_BATCH=100
QUEUE_LONG_RUNNING_CONCURRENT=1
# Set tenants.host when provisioning; API disables queue-*; Workers run queue-schedule
```

**Watch:** MySQL/PG `Threads_connected`, queue pending / `queue:alert-backlog`, `schedule` / `long-running` depth, Redis memory and connections. Near limits, lower `TENANCY_POOL_*` or scale DB/Workers **before** blindly adding API replicas (replicas multiply connection usage).

## Beyond one fleet: regional (cell) deployments

A single fleet targets about **a few thousand active** tenants (see table above). When counts keep growing, prefer **standing up another full stack (region / cell)** instead of endlessly adding API replicas or `tenant_*` databases on one platform DB.

Recommended pattern:

```text
Region A (existing)           Region B (new)
Platform DB A + API/Worker… → Platform DB B + API/Worker…
Existing tenants stay on A    New signups write only to B
```

| Point | Guidance |
|-------|----------|
| One stack per region | Separate platform DB, `TENANCY_DRIVER=database`, API / Worker / Scheduler / Redis, tenant DBs (optionally split further with `tenants.host`) |
| Signup traffic | Point registration/sales at the **current write region** platform/API; from a cutover date, new tenants go only to the new region |
| How tenants return | Region-specific subdomains/apexes, or a thin gateway/DNS map `tenant_code → region API`; do not point old tenant hosts at the wrong region |
| Platform console | Each region has its own `/platform`; this product has **no** cross-region control plane |
| Moves | Avoid moving tenants; cross-region means backup/restore plus DNS/directory changes |
| Cold tenants | `disabled` / archive or DROP the DB while keeping the platform row; do not leave huge numbers of idle-but-`ready` DBs to drag down regional backup / `migrate-all` |

There is **no built-in** `CELL_ID`, global tenant router, or one-click cross-region migrate. Regional scale-out is **ops-level copy of the stack**; tune each cell with the settings in this section. About **~1k tenants per cell** is the comfortable band; grow total capacity as `cells × per-cell capacity`.
