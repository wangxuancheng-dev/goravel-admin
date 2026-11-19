# Goravel package example (feedback)

This repo ships `packages/feedback` as a sample of the official [Package Development](https://www.goravel.dev/digging-deeper/package-development.html) flow.

**Goal:** reuse business features across Goravel apps. Menus/permissions are **not** installed by the package.

## Tenancy

This host injects `OrmQuery(ctx)` via `FeedbackServiceProvider`. Other multi-tenant Goravel apps should inject their own tenant-scoped query; single-DB apps can leave `ResolveQuery` nil.

## Wired in this host

- Providers in `bootstrap/providers.go`
- Routes under `/api/admin/package-demos/feedbacks` (JWT, no Permission middleware for the demo)
- Migration registered in `database/migrations/all.go`

See `packages/feedback/README.md` for APIs and reuse steps.