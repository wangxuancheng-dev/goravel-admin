# Service package split roadmap

`app/services` is still one package with domain-oriented files. Newer security/notification helpers already live in dedicated files (`password_policy.go`, `login_anomaly.go`, `notification_channel.go`) to avoid growing god services.

## Principles

1. **CRUD / code-generator modules** stay in `app/services` to match templates.
2. **Cross-domain orchestration** (paid apply, tenant connection, export/import enqueue) stays as small interfaces + separate files.
3. **Next phase** moves domains into subpackages (`orders`, `payment`, `tenancy`, `rbac`) with thin facades so controller imports stay stable.
4. Prefer splitting **high-coupling / high-LOC** files: `code_generator_service`, `order_service`, `payment_service`, `tenant_connection_service`.
5. **No import cycles**: `services` may import `rbac` / `orders` / `payment` / `tenancy`; those domain packages must **not** import `services`.

## Moved (starter)

| Capability | Package / file | Notes |
|------------|----------------|-------|
| Payment paid status gate | `app/payment` | `PaidResult` + `ApplyPaidResultStatusGate` (pure logic) |
| Payment DB apply | `app/services/payment_apply.go` | `ApplyPaidResult` stays in services; calls `app/payment` gate; alias `services.PaidResult` for compat |
| Data scope / RBAC filters | `app/rbac` | `DataScopeResolved` / `ApplyDataScope` / `CanAccessOwnedBy` / `ResolveAdminDataScope`; `services` keeps same-name thin wrappers |
| Order filters + JSON | `app/orders` | `Filters`, time parsing, shard table mapping, `ToJSON` / `DetailToJSON`; `OrderServiceImpl` stays in services and delegates |
| Import task center | `ImportRecordService` + `ImportController` | list/detail/delete/error download; frontend `export/TaskCenter` |

Next: move gateway drivers + full `ApplyPaidResult` into `app/payment`; moving entire `OrderServiceImpl` (~700 LOC) is deferred.

## Tenancy checklist

Connection-agnostic helpers already live in `app/tenancy` (`MergeTenantHints`, `SubdomainHint`, `CacheKey`, `StoragePrefix`, `PaymentNotifyPath`).

| Still in services | Why |
|-------------------|-----|
| `tenant_connection_service.go` | ORM connection register/migrate/bind; needs facades + DB |
| `tenant_ops_service.go` / `tenant_admin_service.go` / `tenant_scope.go` | Ops CRUD and scoped queries at the generator-style service layer |

When extracting connection later: keep `app/tenancy` free of `services` imports; move connection into e.g. `app/tenancy/connection` and leave thin facades in services.

## Dual frontend

React (`html-react/`) is the **primary shipping UI**; Vue is the parity reference. See [Frontend parity](/en/guide/frontend-parity).
