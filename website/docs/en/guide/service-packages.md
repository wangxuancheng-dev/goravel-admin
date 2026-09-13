# Service package split roadmap

`app/services` stays **one package with domain-oriented files**. Only extract small, pure, testable helpers into `app/*` domain packages when they do not need HTTP/facades orchestration.

## Principles (frozen)

1. **CRUD / code-generator modules** stay in `app/services` to match templates.
2. **Cross-domain orchestration** (`ApplyPaidResult`, tenant connection, export/import enqueue) **stays in services** — do not add more `app` packages or ports layers just for orchestration.
3. **Stop here** for domain packages: keep `payment` / `rbac` / `orders` only. Do **not** move entire OrderService / PaymentService / TenantConnection into new directories.
4. **No import cycles**: `services` may import `rbac` / `orders` / `payment` / `tenancy`; domain packages must **not** import `services`.
5. Default new code to `app/services`; only add a domain package for pure logic, strong reuse, or breaking an import cycle.

## Kept splits (worthwhile)

| Capability | Package / file | Notes |
|------------|----------------|-------|
| Payment paid status gate | `app/payment` | `PaidResult` + `ApplyPaidResultStatusGate` (pure) |
| Payment gateway drivers | `app/payment/gateways/` | one file per channel; `payment.RegisterGateway`; SDK or hand-written; `Notify` returns `PaidResult` only. See [Payments](/en/advanced/payments) §6 |
| Payment DB apply | `app/services/payment_apply.go` | orchestration in services; calls gate; alias `services.PaidResult` |
| Data scope | `app/rbac` | `ApplyDataScope` / `CanAccessOwnedBy`; services call `rbac` directly |
| Order filters + JSON | `app/orders` | Filters / ToJSON; services / controllers call `orders` directly; `OrderServiceImpl` remains in services |
| Tenancy parsing helpers | `app/tenancy` | hints / CacheKey / StoragePrefix (already separate) |
| Import task center | services + controllers | no separate `app/imports` package |

## Explicitly not split / reverted

| Item | Decision |
|------|----------|
| Full `ApplyPaidResult` + `PaymentStore`/`OrderStore` ports | **Do not split**; unfinished `app/payment/store.go` removed |
| Per-vendor `app/<vendor>` payment packages | **Do not split**; stack `payment_gateway_*.go` drivers |
| `tenant_connection_service` / `tenant_ops_service` | **Stay in services** |
| Entire `OrderServiceImpl` / `PaymentService` | **Stay in services** |

## Dual frontend

React (`html-react/`) is the **primary shipping UI**; Vue is the parity reference. See [Frontend parity](/en/guide/frontend-parity).
