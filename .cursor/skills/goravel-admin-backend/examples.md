## Controller skeletons

### Canonical: code generator output
Match `app/services/templates/controller.tpl` and migrated controllers (`article`, `position`, `attachment_category`, `blacklist`, `dictionary`, `permission`, `role`, `user`).

Shared helpers: `app/http/controllers/admin/generated_helpers.go` (`ValidateGeneratedRequest`, `HandleGeneratedServiceError`).

### Generated service side (filters + CRUD)
```go
// services/article_service.go
func BuildArticleFiltersFromHTTP(ctx http.Context) ArticleFilters { /* Input + Query */ }

type ArticleService interface {
    GetByID(id uint) (*models.Article, error)
    GetList(filters ArticleFilters, page, pageSize int) ([]models.Article, int64, error)
    Create(req *admin.ArticleCreate) (*models.Article, error)
    Update(id uint, req *admin.ArticleUpdate) (*models.Article, error)
    Delete(id uint) error
}
```

Business rules (not found, conflicts) return `*errors.BusinessError` from service; controller never calls `ErrorWithLog` for normal CRUD failures.

---

## Legacy patterns (existing modules only — do not use for new CRUD)

These appear in older hand-written controllers (admin, payment_method, …). Keep when editing those files; do not copy for new modules.

### Legacy: `response.FindByID` in controller
```go
x, resp := response.FindByID[models.X](ctx, id, &response.FindByIDOptions{...})
if resp != nil { return resp }
```

### Legacy: partial update via `Request().All()`
Controller loads model, mutates fields present in request body, saves via service.

### Legacy: `response.Paginate` shortcut (export list, online admins)
```go
return response.Paginate(ctx, list, total, page, pageSize)
```

### Tree endpoints (menu, department)
Menu returns `menus` + `list`; department returns `list` only (flat when search filters present). CRUD uses `ValidateGeneratedRequest` / `HandleGeneratedServiceError`; tree/filter logic lives in the service.

---

## Auth-style patterns (hand-written)

### Login with token header
```go
return response.SuccessWithHeader(ctx, "login_success", "Authorization", "Bearer "+token, http.Json{
    "token": token, "user": http.Json{"id": user.ID},
})
```

### Auth status choices
- Invalid credentials → 401 `username_or_password_error`
- Disabled account → 403 `account_disabled`
- Not logged in → 401 `not_logged_in`

Reference: `app/http/controllers/admin/auth_controller.go`, `app/http/controllers/api/auth_controller.go`
