# Schedule demos (activity windows + order expire)

> Open-source samples, not a production campaign/acquiring stack. HTTP APIs require `DevelopmentOnly` / `APP_ENABLE_DEV_TOOL`.

| Scenario | Approach | Precision |
|----------|----------|-----------|
| Activity once / daily window | `demo_activities` + `activity:sync-status` (every 10s) + live `IsActive` | Gate ~ clock skew; stored `status` may lag ~10s |
| Unpaid order auto-cancel | `orders.expire_at` + Delay job + `order:cancel-expired` every minute | Usually seconds; pay path re-checks `expire_at` |

Full request examples: see the [Chinese version](/advanced/scheduled-demos).

Admin UI: System → Demo Activities. Toggle with `MODULE_SCHEDULE_DEMO_ENABLED` (default true). API: `/api/admin/demo-activities`.

## Configurable cron (whitelist)

Per-tenant whitelist handlers on landlord `flexible_schedules` (unique handler+tenant_id, optional payload JSON). Tenant backup stays on kernel `tenant:backup-scheduled`. Tick: `flexible-schedule:tick` every minute. See Chinese doc for seed/permission notes.

