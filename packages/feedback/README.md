# feedback (example Goravel package)

Demo package for **reusable Goravel modules** without installing menus/permissions.

Official package docs: https://www.goravel.dev/zh_CN/digging-deeper/package-development.html

## What it provides

| Piece | Path |
|-------|------|
| Model | `models/feedback.go` → table `feedbacks` |
| Migration | `migrations/20261119000000_create_feedbacks_table.go` |
| Service | `NewService()` / `contracts.Service` |
| HTTP | `http.RegisterAdminRoutes` → `/api/admin/package-demos/feedbacks` |
| Tenancy hook | `ResolveQuery` / `ResolveActorID` (host injects) |

## Host wiring (this repo)

1. `feedback.ServiceProvider` + `providers.FeedbackServiceProvider` in `bootstrap/providers.go`
2. Migration registered in `database/migrations/all.go` (runs on tenant migrate)
3. Routes mounted on the logged-in admin group (JWT, **no** Permission middleware)

```go
feedbackpkg.ResolveQuery = func(ctx context.Context) orm.Query {
    return appfacades.OrmQuery(ctx) // tenant-safe
}
```

## Use in another Goravel app

```bash
# from that app
go get / path to this module or publish packages/feedback as its own module
./artisan package:install <module-path>   # if setup/setup.go is configured
# or manually: register ServiceProvider, copy migration, call RegisterAdminRoutes
```

Set `ResolveQuery` to that app's tenant-aware query (or leave nil for single-DB `facades.Orm().Query()`).

## API (this host)

- `GET    /api/admin/package-demos/feedbacks`
- `POST   /api/admin/package-demos/feedbacks`  body: `{ "contact", "content" }`
- `GET    /api/admin/package-demos/feedbacks/{id}`
- `POST   /api/admin/package-demos/feedbacks/{id}/done`
- `DELETE /api/admin/package-demos/feedbacks/{id}`

Menus and RBAC stay in the host app on purpose.
