# goravel-admin — Project Extensions

This file extends the official `goravel-development` skill with rules specific
to **goravel-admin**. AI agents MUST read this file together with `SKILL.md`.

## Project identity

- **Repo:** goravel-admin — a full-featured admin backend (not goravel-lite).
- **Go module:** `goravel` — import app code as `goravel/app/...`, `goravel/config`, etc.
- **Framework:** `github.com/goravel/framework v1.18.x` (full scaffold; all facades installed).
- **HTTP driver:** Gin via `github.com/goravel/gin`.
- **Facades:** `app/facades/` — see `website/docs/reference/facades.md` for install/uninstall notes.

## Skill layering (which skill to use)

| Task | Read |
|------|------|
| Goravel framework basics (routing, ORM, migrations, `make:*`, tests, facades) | `.agents/skills/goravel-development/SKILL.md` (this tree) |
| Admin API responses, BusinessError, controllers, services, auth/permissions | `.cursor/skills/goravel-admin-backend/SKILL.md` |
| Vue 3 admin frontend (`html/`) | `.cursor/skills/goravel-admin-frontend/SKILL.md` |
| React 19 admin frontend (`html-react/`) | `.cursor/skills/goravel-admin-frontend-react/SKILL.md` |

Framework conventions live here; **business/admin conventions live in `.cursor/skills/`**.
Do not invent admin response shapes or frontend patterns — follow the matching Cursor skill.

## Layout differences from a vanilla Goravel app

```
app/
  http/controllers/admin/   # Admin JSON API controllers
  http/controllers/api/     # Public/mobile API controllers
  http/middleware/          # Auth, permission, locale, etc.
  http/requests/            # Form request validation structs
  http/response/            # Canonical JSON response helpers (admin-specific)
  services/                 # Business logic (prefer over fat controllers)
  models/                   # GORM models
  errors/                   # BusinessError definitions
  providers/                # App service providers (routes, auth, queue, …)
routes/
  admin.go                  # /admin/* routes
  api.go                    # /api/* routes
  web.go                    # Web/static routes
html/                       # Vue 3 + Element Plus admin SPA
html-react/                 # React 19 + Ant Design admin SPA
website/                    # VitePress docs (zh/en)
config/                     # Goravel config (one file per concern)
lang/                       # Backend i18n (zh-CN, en-US, …)
```

## Bootstrap & routing

- Entry: `main.go` → `bootstrap.Boot().Start()`.
- Routes registered in `bootstrap/app.go`: `routes.Web()`, `routes.Api()`, `routes.Admin()`, `routes.Pprof()`.
- Admin routes use auth + permission middleware; match existing controllers in the same module.

## Admin API rules (summary — details in Cursor backend skill)

- All admin JSON responses go through `app/http/response/response.go` helpers.
- Success: `{ code: 200, message, data?, trace_id? }`.
- Error: `{ code, message, error_code, trace_id?, errors? }`.
- **New CRUD:** follow code generator output — `app/services/templates/controller.tpl`, example `article_controller.go`. Use `ValidateGeneratedRequest`, `HandleGeneratedServiceError` (in `generated_helpers.go`), service-layer `Create/Update/Delete`, pagination key `list`.
- **Migrated CRUD:** article, position, attachment_category, blacklist, dictionary, permission, role, user (+ user_balance_log list/stats).
- **Legacy modules** (menu, department, admin, payment, order, …): may use `FindByID`, per-action `ErrorWithLog`, `data` pagination — match in place when editing only.
- Business errors from service layer; reuse codes from `app/errors/errors.go`.

## Frontend (dual stack)

- **Vue:** `html/src/` — axios `utils/request.js`, `apiFactory`, `useListPage`.
- **React:** `html-react/src/` — `utils/request.ts`, `createCRUDApi`, `useListPage`, `SimpleCrudPage`.
- Both expect the same backend response contract (`code === 200` for success).
- When adding a CRUD module, prefer **Dev → Code Generator** (backend + frontend), or match `article_controller.go` manually.

## AI integration

- Goravel AI SDK is configured in `config/ai.go` with `github.com/goravel/openai`.
- Env: `AI_ENABLED`, `AI_PROVIDER`, `AI_API_KEY`, `AI_MODEL`, optional `AI_BASE_URL`.
- Current usage: code generator AI assist (`app/services/ai_service.go` → `GenerateWithAI`).

## Custom drivers & packages

- DM database: `github.com/wangxuancheng-dev/goravel-dm` (see `website/docs/reference/drivers.md`).
- Queue drivers: `goravel-kafka`, `goravel-nsq`, `goravel-rabbitmq`, `goravel-redis-stream` (see `website/docs/reference/drivers.md`).
- Prefer existing driver patterns when adding new integrations.

## Commands (this project)

```shell
go run . artisan list              # discover commands
go run . artisan migrate           # run migrations
go run . artisan db:seed           # seed data
go test ./path/to/pkg -run TestName   # prefer targeted tests
```

## Do not

- Rename default bootstrap paths without updating `WithPaths()`.
- Hand-write boilerplate that `make:*` can generate — run `go run . artisan list` first.
- Duplicate admin response or frontend conventions in this file — link to `.cursor/skills/` instead.
- Modify `SKILL.md` in this directory — it tracks goravel-lite upstream; put project-only deltas here in `CUSTOM.md`.
