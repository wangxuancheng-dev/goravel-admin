## Module names for `ErrorWithLog` (reuse these)
Prefer reusing existing module strings so logs are grouped consistently.

Observed in admin controllers:
- `auth`, `captcha`
- `admin`, `online_admin`
- `user`, `password`
- `menu`, `role`, `permission`, `department`, `position`
- `dictionary`, `blacklist`
- `article`, `notification`
- `order`, `payment`, `payment_method`
- `attachment`, `attachment_category`
- `export`, `import`
- `config`, `observability`
- `login-log`, `operation-log`, `system-log`
- `menu-test` (debug/test)

Guideline:
- If a module already exists for the domain, **do not invent a new one**.
- If adding a new domain, prefer **kebab-case** for new module names and keep it stable:
  - ✅ kebab-case: `login-log`, `operation-log`, `system-log`, `online-admin`
  - ⚠️ existing snake_case modules exist (`payment_method`, `online_admin`, `attachment_category`) — keep them as-is, but don’t introduce more snake_case.
  - Rule of thumb: use the same word separator style as the closest existing module; otherwise default to kebab-case.

## What to put in `attrs` (safe, useful, consistent)
`ErrorWithLog` will auto-add `error` if you don’t set it. Provide only small, non-sensitive context.

Good attrs:
- identifiers: `id`, `<resource>_id`, `user_id`, `admin_id`, `role_id`, `menu_id`, `order_id`
- lookup keys: `username`, `slug`, `email`, `phone`, `trace_id`
- action context: `action`, `status`, `type`, `filters`
- pagination: `page`, `page_size`

Avoid attrs:
- passwords, tokens, Authorization headers
- full request bodies (too big + may contain secrets)
- payment gateway secrets / API keys in config maps

## Decision table (which response helper to use)

### Generated CRUD (canonical)
- Validation: `validateGeneratedRequest(ctx, &req)`
- Service errors: `handleGeneratedServiceError(ctx, status, err)` — maps BusinessError codes to HTTP status
- Success list: `response.Success` with `list`, `total`, `page`, `page_size`
- Delete success: `response.Success(ctx, "delete_success", http.Json{})`

### Hand-written / legacy modules
- Validation: inline `ValidateRequest` → `ValidationError`
- Business failure: `response.Error(ctx, 4xx, err)` or `businessErr.Code`
- Unexpected infra: `response.ErrorWithLog(ctx, module, err, attrs)`
- Paginated list: manual Success, or `response.Paginate` (export, online admins)
- ORM load in controller: `response.FindByID` (legacy modules only)

## Response payload key conventions (observed)

### Paginated lists
Always include: `total`, `page`, `page_size`.

Rows array key: **`list`** for all admin list endpoints (including payment / order / payment_method).

Frontends still accept legacy `data` via `normalizeListResponse` for compatibility, but new/changed controllers must return `list`.

### Detail endpoints
Singular resource key: `user`, `role`, `position`, `payment_method`, `order`, etc.

### Tree endpoints
- Menu: returns both `menus` and `list` (compat)
- Department: returns `list` only (tree or flat search results)

### Lookup endpoints
May use plural nouns, e.g. `dictionaries`, `types`

## Controller structure

### Canonical (new CRUD — code generator)
- Template: `app/services/templates/controller.tpl`
- Example: `app/http/controllers/admin/article_controller.go`
- Migrated: `article`, `position`, `attachment_category`, `blacklist`, `dictionary`, `permission`, `role`, `user` (CRUD only; `UpdateBalance`, `ResetPassword`, `Export` remain hand-written)
- Helpers: `ValidateGeneratedRequest`, `HandleGeneratedServiceError` in `app/http/controllers/admin/generated_helpers.go`
- Service: `BuildXFiltersFromHTTP`, `GetByID`, `GetList`, `Create`, `Update`, `Delete`
- Pagination: always `list` + `total` + `page` + `page_size`
- Destroy: `Success(ctx, "delete_success", http.Json{})`

### Legacy (hand-written older modules — edit in place only)
| Pattern | Where |
|---------|-------|
| Fat orchestration in controller | attachment upload/stream, pprof, SSE; async export uses shared `EnqueueAsyncExport` |
| `response.Paginate` | export list, online admins |
| apidoc/swag annotations | admin, payment_method (optional) |

Do **not** apply legacy patterns to new modules — use code generator output instead.

Menu / department / admin / payment_method / payment Index-Show / order CRUD helpers / attachment Index / online_admin / config / export CRUD / notification / dashboard / observability (trace+audit) / schedule are migrated toward thin controllers + service ownership.
Keep hand-written: monitor, WS, codegen, form demo, pprof sampling, and export/import queue orchestration.

## Copy/paste examples

### Example: standard request validation
```go
var req adminrequests.MenuCreate
errors, err := ctx.Request().ValidateRequest(&req)
if err != nil {
    return response.Error(ctx, http.StatusBadRequest, err.Error())
}
if errors != nil {
    return response.ValidationError(ctx, http.StatusBadRequest, "validation_failed", errors.All())
}
```

### Example: logging unexpected errors safely
```go
if err := facades.Orm().Query().Save(&model); err != nil {
    return response.ErrorWithLog(ctx, "menu", err, map[string]any{
        "menu_id": model.ID,
        "slug":    model.Slug,
    })
}
```

### Example: BusinessError with placeholders
```go
// service layer:
return apperrors.ErrInsufficientBalance.WithParams(map[string]any{"balance": currentBalance})

// controller boundary (preserves WithParams formatting):
if err != nil {
    return response.Error(ctx, http.StatusBadRequest, err)
}
```
