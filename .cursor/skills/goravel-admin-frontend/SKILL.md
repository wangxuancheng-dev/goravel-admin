---
name: goravel-admin-frontend
description: Implements the Vue 3 + Vite admin frontend in html/ using the existing axios request wrapper (src/utils/request.js), API factory (src/utils/apiFactory.js), and list-page composables (useListPage/useStandardListPage). Use when creating pages/components under html/, wiring Vue APIs, handling auth tokens, i18n messages, and matching backend response conventions. For React admin work under html-react/, use goravel-admin-frontend-react instead.
---

# Goravel Admin Frontend (html/)

## Project basics (assume by default)
- Tech stack: Vue 3, Vite, TypeScript, Element Plus, Pinia, Vue Router, vue-i18n.
- Source root: `html/src` with alias `@` configured in `html/vite.config.js`.
- API modules live in `html/src/api/*.js` and call `request` from `html/src/utils/request.js`.
- **Dual frontend (required sync):** Vue and React are both supported admin UIs. New/changed admin features must be mirrored in `html-react/` in the same change set so neither side lags. Prefer matching React patterns when both exist; for React-only work under `html-react/`, use `goravel-admin-frontend-react`, then sync Vue here.

## Backend response contract (must match)
The frontend request layer expects backend responses shaped like:
- success: `{ code: 200, message: string, data?: any, trace_id?: string }`
- error: `{ code: <http_status>, message: string, error_code: string, trace_id?: string, errors?: object }`

Do not invent new response shapes in frontend code. Handle the contract via the existing wrapper.

## HTTP client conventions (must follow)
Use `html/src/utils/request.js` (axios instance) for all API calls.

### Auth token + headers
- Token storage key: `token` (via `Storage.getItem/setItem`)
- Request header: `Authorization: Bearer <token>`
- Response may refresh token via:
  - response header `authorization` (Bearer ...)
  - or payload `res.data.token`

### i18n + locale headers
- Request sets `Accept-Language` based on `i18n.global.locale` (zh-CN / en-US).

### Timezone header
- Request sets `X-Timezone` from app store or stored timezone, falling back to browser timezone.

### Error handling behavior
The response interceptor already:
- Treats `res.code !== 200` as a business error even if HTTP status is 200.
- Extracts `message`, `error_code`, `code` and tries to translate message by `error_code`:
  - `common.<error_code>` first (e.g. `common.operation_failed`)
  - then `messages.<error_code>`
- Handles 401 (logout + redirect /login) and 403 (debounced message) globally, except for auth endpoints (`/login` `/logout`) which are delegated to the page (e.g. Login view).
- Special-cases 429 for export endpoints: don’t toast automatically; let business code handle it.

Therefore:
- Do not duplicate global toasts in pages when `err.__handled === true`.
- For login/logout pages, expect errors to be unhandled and display them locally.

### Paginated list responses
Backend returns rows under `res.data.list` for admin list endpoints (code generator and migrated modules).
`normalizeListResponse` / list helpers still accept legacy `data` for compatibility; **new work must use `list`**.
`useStandardListPage` handles both keys.

## API layer conventions
Prefer following the existing pattern used by modules like `html/src/api/user.js`:
- build a base CRUD API with `createCRUDApi(resource)`
- add non-CRUD endpoints via `extendApi(baseApi, { ... })`
- export functions via destructuring with names like `getXList/getXDetail/createX/updateX/deleteX`

Extend with custom methods via `extendApi(baseApi, customMethods)` when needed.

When wiring a page:
- Prefer `useStandardListPage()` / `useStandardTreeListPage()` for CRUD list pages.
- Keep API calls in `html/src/api/...` modules (do not call `request` directly from views unless it’s truly one-off).

## Output expectations (how the agent should report changes)
When implementing frontend changes, always include:
- Changed files (path + purpose)
- API endpoints touched (method + path) and expected response keys used
- UX behavior for errors (what’s global vs what the page shows)
- Test plan (commands + manual steps)

## Additional resources
- For module patterns, API templates, and page wiring examples, see [examples.md](examples.md).
- For response/error-code mapping and i18n keys, see [reference.md](reference.md).
- Dual-frontend checklist: `website/docs/guide/frontend-parity.md`
- Project docs site: `website/` (VitePress; not repo-root `docs/`)
