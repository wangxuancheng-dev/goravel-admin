---
name: goravel-admin-backend
description: Implements backend features in this Goravel admin project using code-generator CRUD patterns (article_controller), HTTP response helpers (response.Success/Error/ValidationError/ErrorWithLog), BusinessError with i18n, and service-layer logic. Use when adding or modifying controllers, requests, services, auth/permissions, exports, uploads, or fixing API response/error handling.
---

# Goravel Admin Backend

## Canonical response format (must match)
All API responses are JSON and use the helpers in `app/http/response/response.go`.

## Additional resources
- For module names, logging attrs, and legacy notes, see [reference.md](reference.md).
- For controller/service skeletons, see [examples.md](examples.md) — **code generator output is the source of truth**.

### Success JSON
- `code`: `200`
- `message`: translated via `trans.Get(ctx, messageKey)` (default `"success"`)
- `data`: optional
- `trace_id`: auto-included when present in context

Use:
- `response.Success(ctx)` / `response.Success(ctx, data)` / `response.Success(ctx, "message_key", data)`
- `response.SuccessWithHeader(ctx, "message_key", headerKey, headerValue, data)`
- Manual paginated `response.Success` with `list` + `total`/`page`/`page_size` (code generator default)

### Error JSON
- `code`: HTTP status code (and also the JSON `code` field)
- `message`: user-facing message (translated)
- `error_code`: stable string key for frontend branching
- `trace_id`: auto-included when present in context

Use:
- `response.Error(ctx, httpStatus, messageKeyOrErr)`
- `response.ErrorWithLog(ctx, ...)` for unexpected infra failures in **hand-written** modules that are not yet on the shared helper (e.g. `notification_ws`)
- Generated / migrated controllers use `HandleGeneratedServiceError` (CRUD **and** Export in `controller.tpl`)

### Validation error JSON
- same as Error JSON plus `errors` field map and first-field message when available

Use:
- `response.ValidationError(ctx, http.StatusBadRequest, "validation_failed", errors.All())`
- Generated CRUD: `validateGeneratedRequest(ctx, &req)`

## Error model (BusinessError + i18n)
Business errors live in `app/errors/errors.go`. Services return `*errors.BusinessError` for expected failures; controllers map them via `handleGeneratedServiceError` (generated) or `response.Error` (hand-written).

Rules:
- Stable snake_case codes; reuse existing codes before adding new ones
- Use `WithParams(...)` for dynamic message placeholders (`{key}` / `${key}`)
- Service layer owns business rules; controller stays thin

## New CRUD modules — follow the code generator (canonical)

**Source of truth:** `app/services/templates/controller.tpl` → e.g. `app/http/controllers/admin/article_controller.go`.

When adding a new admin CRUD module, **match the generator output** (or use Dev → Code Generator). Do **not** copy legacy patterns from admin/payment unless you are fixing those files in place.

### Generated controller helpers (shared in admin package)
All generated / migrated CRUD controllers use helpers in `app/http/controllers/admin/generated_helpers.go`:

```go
ValidateGeneratedRequest(ctx, &req)
HandleGeneratedServiceError(ctx, "module_name", http.StatusInternalServerError, err, attrs)
```

`HandleGeneratedServiceError`:
- maps BusinessError codes to appropriate HTTP status (`*_not_found` → 404, `*_exists` / `*_already_exists` → 400, `role_protected_*` → 403, …)
- logs unexpected failures (500+) via `ErrorWithLog`

Do **not** duplicate these per controller file.

### Migrated modules (canonical CRUD pattern)
These admin modules follow the code generator pattern end-to-end:

| Module | Notes |
|--------|-------|
| `article` | Reference implementation |
| `position`, `attachment_category`, `blacklist`, `dictionary`, `permission`, `role` | Full CRUD |
| `menu`, `department` | Tree/index special cases kept; CRUD uses shared helpers |
| `admin` | CRUD migrated; Export / 2FA actions remain hand-written |
| `payment_method` | CRUD via `PaymentMethodService` (split from payment god service) |
| `password` | UpdateOwnPassword / ResetPassword in AdminService + shared helpers |
| `login-log`, `operation-log`, `system-log` | List/Show/Delete/Batch/Clean migrated; title/module options in service |
| `user` | CRUD + UpdateBalance/ResetPassword/Export use shared helper |
| `user_balance_log` | List/statistics use shared error helper; `Store` remains hand-written (sharding) |
| `payment` | Index/Show thin + `PaymentToJSON` + `BuildPaymentFiltersFromHTTP`; gateway is `PaymentGatewayService`; export/queue remain hand-written |
| `order` | Index/Show/Store/Update/Destroy thin + `BuildOrderFiltersFromHTTP` + `OrderCreate`/`OrderUpdate` requests; Import via `ImportUploadedCSV`; export orchestration uses shared error helper |
| `attachment` | Index filters/`AttachmentToJSON` + shared error helper; upload/chunk/stream stay HTTP-layer |
| `online_admin` | Full list/kick via `online_admin_service` |
| `config` | GetByGroup/Save/TestEmail via `config_service` |
| `export` | Index/Destroy/BatchDestroy via filters/`ExportRecordToJSON`/`DeleteWithFile`; Download/SSE stay HTTP-layer |
| `notification` | List/mark/Store use shared helper; `PrepareAnnouncementContent` / `CreateAnnouncement` in service |
| `dashboard` | Stats via `dashboard_service`; SSE loop stays in controller |
| `observability` | TraceAggregate/AuditTimeline via `observability_service`; SlowSQL/API use shared helper; pprof stays HTTP-layer |
| `schedule` | Index/Run use shared helper (`schedule_busy` → 409) |
| `ai_lab` | Unexpected errors use shared helper; multipart stays in controller |

**Do not migrate** (edit in place only): `auth` (login flow), `monitor` (ops sysinfo), `notification_ws`, `code_generator`, `form_demo`, payment/order/user export queue orchestration internals, attachment Download/Preview streaming, …

**Sharding queries:** order/payment/balance_log already share `ShardingQueryService` (`docs/SHARDING_QUERY_SERVICE.md`). Do not invent a parallel UNION layer unless a third pattern diverges.

### Generated controller structure
```go
func (c *XController) buildXFilters(ctx http.Context) services.XFilters {
    return services.BuildXFiltersFromHTTP(ctx)  // filters live in service package
}
func (c *XController) XService(ctx http.Context) services.XService {
    return services.NewXService(ctx)
}
```

| Action | Pattern |
|--------|---------|
| Index | `GetList(filters, page, pageSize)` → `{ list, total, page, page_size }` |
| Show | `GetByID(id)` → `{ "<module>": item }` |
| Store | `validateGeneratedRequest` → `Create(&req)` → `{ "<module>": item }` |
| Update | `validateGeneratedRequest` → `Update(id, &req)` → `{ "<module>": item }` |
| Destroy | `Delete(id)` → `Success(ctx, "delete_success", http.Json{})` |
| Export | async queue or sync `response.Export` (when enabled in generator) |

Service interface (generated): `GetByID`, `GetList`, `Create`, `Update`, `Delete` — business logic and ORM queries stay in service.

### Pagination
New generated modules always use `"list"` for the rows array. Payment / order / payment_method also use `"list"` (legacy `"data"` is no longer emitted).

## Hand-written modules (auth, export, config, …)

Older/system modules may still use:
- `response.FindByID` directly in controller
- Partial update via `ctx.Request().All()` in controller
- `response.Paginate` shortcut (export list, online admins)

Async export orchestration uses `EnqueueAsyncExport` in `app/http/controllers/admin/async_export.go` (hand-written modules + `controller.tpl`). Controllers only build filters and pass the job instance.

## HTTP status mapping (default)
- **400** — validation, params_error, business conflicts
- **401** — not logged in, bad credentials
- **403** — forbidden, disabled account
- **404** — not found (`*_not_found`, `record_not_found`)
- **500** — unexpected failures (ErrorWithLog in hand-written modules; handleGeneratedServiceError in generated)

## Deliverable expectations
When implementing changes, include:
- Changed files + purpose
- API success/error shapes and status codes
- New error codes (if any) and why not reused
- Test plan (targeted `go test` + manual requests)

## References
- `docs/TENANT_RESERVED.md` — multi-tenant (off vs database-per-tenant); prefer `OrmQuery(ctx)` / `tenancy.CacheKey`
- `app/services/templates/controller.tpl` — generator template (canonical)
- `app/http/controllers/admin/generated_helpers.go` — shared Validate/HandleGeneratedServiceError
- `app/http/controllers/admin/article_controller.go` — latest generated example
- `app/http/response/response.go` — response helpers
- `app/errors/errors.go` — BusinessError
