# Service code layout

Put business logic in `app/services` by default (one package, domain-oriented files). Extract into `app/*` domain packages only for small, pure, testable helpers that do not need HTTP/Facades orchestration.

## Conventions

1. **CRUD / code-generator modules** live in `app/services` to match templates.
2. **Cross-domain orchestration** (payment apply, tenant connection, export/import enqueue) stays in `app/services` — do not add a ports layer just for orchestration.
3. **Existing domain packages**: `app/payment`, `app/search`, `app/rbac`, `app/tenancy`. Do not create `app/<domain>` by default (e.g. no `app/notice` for a generated Notice module).
4. **Import direction**: `services` may import domain packages; domain packages must **not** import `services`.
5. Prefer `app/services` for new code; add a domain package only for pure logic, strong reuse, or to break an import cycle.

## Domain packages

| Capability | Location | Notes |
|------------|----------|-------|
| Payment status gate | `app/payment` | `PaidResult`, `ApplyPaidResultStatusGate` |
| Payment gateway drivers | `app/payment/gateways/` | one file per channel; `payment.RegisterGateway`; see [Payments](/en/advanced/payments) |
| Payment wiring | `providers.PaymentServiceProvider` | blank-import gateways; inject `ResolvePaymentAmount` |
| Payment DB apply | `app/services/payment_apply.go` | orchestration in services; allowlist `EnabledPaymentGateways` |
| Search engine | `app/search` | `Engine` / `Resolve` / drivers; must not import services |
| Order search adapter | `app/search/orders` | `Document` / `Push` / `Query`; loaders via SearchServiceProvider |
| Data scope | `app/rbac` | `ApplyDataScope` / `CanAccessOwnedBy` |
| Order filters + JSON | `app/services/order_filters.go`, `order_json.go` | same package as CRUD |
| Tenancy helpers | `app/tenancy` | hints / CacheKey / StoragePrefix |
| Import task center | services + controllers | no separate `app/imports` |

When extracting a domain package later: move the module and keep the matching Provider as the blank-import / hook injection point; domain packages must not import `app/services`.

## Dual frontend

React (`html-react/`) is the primary UI; Vue (`html/`) is the parity implementation. See [Frontend parity](/en/guide/frontend-parity).
