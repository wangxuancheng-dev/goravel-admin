---
name: goravel-admin-frontend-react
description: Implements the React 19 + Vite admin frontend in html-react/ using request.ts, apiFactory, useListPage, SimpleCrudPage, and List+FormModal+config patterns. Use when creating or modifying React admin pages, APIs, permissions, i18n, or matching backend response conventions.
---

# Goravel Admin Frontend (html-react/)

## Project basics
- Tech stack: React 19, Vite, TypeScript, Ant Design 6, Zustand, React Router 7, react-i18next.
- Source root: `html-react/src` with alias `@`.
- API modules: `html-react/src/api/*.ts` via `request` from `utils/request.ts`.
- **Dual frontend (required sync):** Admin features must ship in **both** `html-react/` and `html/` in the same change set. Prefer implementing/adjusting the React page first as the pattern reference, then mirror to Vue so Vue users never miss the feature. Do not ship React-only or Vue-only admin features unless the user explicitly scopes one frontend.

## Backend response contract (must match)
- success: `{ code: 200, message: string, data?: any, trace_id?: string }`
- error: `{ code: <http_status>, message: string, error_code: string, trace_id?: string, errors?: object }`

Do not invent new response shapes. Use the existing request wrapper.

### Paginated list responses
Backend returns rows under `data.list` for **all new code-generated CRUD** (e.g. article).
Legacy payment/order modules use `data.data` instead.
Always use `normalizeListResponse()` in API modules — it handles both keys.

## HTTP client
Use `html-react/src/utils/request.ts` for all API calls.

- Token: `Storage` key `token`; header `Authorization: Bearer <token>`
- Locale: `Accept-Language` from i18n (`zh-CN` / `en-US`)
- Timezone: `X-Timezone` from app store / storage / browser
- Global handling: business `code !== 200`, 401 logout, 403 toast, export 429 special-cased
- If `err.__handled === true`, do not duplicate toasts in the page

## API layer
Prefer `createCRUDApi` + `extendApi`:

```ts
import { createCRUDApi } from '@/utils/apiFactory'
import { normalizeListResponse } from '@/utils/normalize'

const api = createCRUDApi('positions')

export async function getPositionList(params?: Record<string, unknown>) {
  return normalizeListResponse(await api.list(params))
}

export const getPositionDetail = api.detail
export const createPosition = api.create
export const updatePosition = api.update
export const deletePosition = api.delete
```

List APIs must be compatible with `ListFetchFn` (`Promise<ApiResponse<PaginatedData>>`) so they plug into `useListPage` without casts.

## List page paradigms (pick one)

### A) Simple CRUD → `SimpleCrudPage`
Use when fields are flat (input / textarea / number / status / password), modal form is enough, no heavy custom UI.

Examples: `pages/position/PositionList.tsx`, `pages/dictionary/DictionaryList.tsx`, `pages/permission/PermissionList.tsx`, `pages/blacklist/BlacklistList.tsx`.

### B) Complex CRUD → List + FormModal + `*.config.ts`
Use when the page needs custom columns, exports, status switches, nested relations, rich editors, multi-modals, etc.

File layout:

```
pages/<module>/
  <Module>List.tsx          # table + search + toolbar
  <Module>FormModal.tsx     # create/edit (when non-trivial)
  <module>.config.ts        # initialSearchForm, Row type, transformRow, helpers
```

Examples: `pages/article/`, `pages/admin/`, `pages/role/`, `pages/order/`, `pages/payment/` (PaymentList, PaymentMethodList).

Wire the list with:

```ts
const { tableData, loading, searchForm, onSearchFormChange, handleSearch, handleReset, ... } =
  useListPage<Row, typeof initialSearchForm>({
    fetchApi: getXList,
    initialSearchForm,
    normalizeRows: true,
    transformData: transformXRow,
  })
```

- Prefer `onSearchFormChange` for `SearchForm` (no `as never`).
- Put search defaults / row mapping in `*.config.ts`, not inline in the List file when non-trivial.

### Do not mix
Do not hand-roll a near-duplicate of `SimpleCrudPage` for simple modules. Do not force complex modules into `SimpleCrudPage`.

## Permissions
- Buttons: `PermissionButton` + `usePermission().getButtonState('resource.action')`
- Slugs match backend: `article.store`, `admin.update`, `role.destroy`, etc.
- Create toolbar via `useCrudActions({ createPermission, deleteApi, onCreate, onRefresh })`

## i18n
- Locales: `src/i18n/locales/zh-CN.json`, `en-US.json`
- Menu titles: slug kebab-case from backend → snake_case keys under `menu.*` via `utils/menuTitle.ts`
- Prefer existing keys (`common.*`, `table.*`, module namespaces)

## Output expectations
When implementing React frontend changes, include:
- Changed files (path + purpose)
- API endpoints touched and response keys used
- Which list paradigm was chosen (SimpleCrudPage vs List+FormModal+config)
- Error UX (global vs page-local)
- Test plan (`npm run type-check` + manual steps)

## Additional resources
- Patterns: [examples.md](examples.md)
- Errors / env: [reference.md](reference.md)
- Project overview: `html-react/README.md`
