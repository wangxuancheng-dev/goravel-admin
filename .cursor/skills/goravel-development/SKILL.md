---
name: goravel-development
description: >
 Use when writing or modifying Goravel framework code in this repo: routing,
 controllers, facades, configuration, ORM, migrations, console commands,
 and tests. For admin API/frontend business conventions, use goravel-admin-*
 skills instead.
---

# Goravel Development (goravel-admin)

This project uses the **official Goravel framework skill** plus a local extension.

## Required reading

Before changing Goravel framework code, read and apply **both** files:

1. [.agents/skills/goravel-development/SKILL.md](../../.agents/skills/goravel-development/SKILL.md) — upstream Goravel conventions (facades, `make:*`, ORM, testing)
2. [.agents/skills/goravel-development/CUSTOM.md](../../.agents/skills/goravel-development/CUSTOM.md) — goravel-admin project layout and skill routing

Do not modify `SKILL.md` in `.agents/` when updating project rules; edit `CUSTOM.md` instead.

## Skill layering

| Task | Skill |
|------|-------|
| Framework: routing, ORM, migrations, facades, `make:*`, tests | `.agents/skills/goravel-development/` (this entry) |
| Admin API: responses, BusinessError, auth, services | `.cursor/skills/goravel-admin-backend/` |
| Vue frontend (`html/`) | `.cursor/skills/goravel-admin-frontend/` |
| React frontend (`html-react/`) | `.cursor/skills/goravel-admin-frontend-react/` |

## Quick project facts

- Go module: `goravel`; full scaffold (not lite); framework `v1.18.x`
- Routes: `routes/admin.go`, `routes/api.go`; controllers under `app/http/controllers/`
- Admin JSON helpers: `app/http/response/response.go`
- Docs: VitePress under `website/` (`website/docs/…`); root `docs/` is Swagger only
- Upstream skill source: [goravel/goravel-lite](https://github.com/goravel/goravel-lite) — upgrade by copying `.agents/skills/goravel-development/SKILL.md` from upstream and merging `CUSTOM.md` as needed
