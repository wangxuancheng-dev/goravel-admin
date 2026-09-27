## Critical files
- `html-react/src/utils/request.ts` — axios interceptors, token / locale / timezone, global errors
- `html-react/src/utils/apiFactory.ts` — `createCRUDApi`, `extendApi`
- `html-react/src/hooks/useListPage.ts` — paginated lists (`ListFetchFn`, `onSearchFormChange`)
- `html-react/src/components/SimpleCrudPage.tsx` — simple CRUD shell
- `html-react/src/utils/menuTitle.ts` — menu slug → i18n title

## Env / routing
- API base: `VITE_API_BASE_URL` + `VITE_API_PREFIX` (default `/api/admin`)
- Dev server default port `3008` (see `vite.config`)
- SPA hosting needs fallback to `index.html` on refresh

## Error contract
1. HTTP error (`error.response.status`)
   - 401: logout + redirect (except auth pages)
   - 403: debounced toast
   - 429: export endpoints left to page logic
2. HTTP 200 but `res.code !== 200`
   - Rejected with `code`, `errorCode`, translated `message`, often `__handled: true`

## i18n for `error_code`
Prefer `common.<error_code>`, then `messages.<error_code>`.

## List typing notes
- `ListFetchFn` returns `ApiResponse<PaginatedData>` (row unknown)
- `useListPage<T>` refines rows via `transformData?: (row: Record<string, unknown>) => T`
- Avoid `as never` on `fetchApi` / SearchForm `onChange`
- **Rows key**: backend may use `data.list` or `data.data`; `normalizeListResponse` in `utils/normalize.ts` handles both — always call it in list API wrappers

## When to choose which page pattern
| Situation | Pattern |
|---|---|
| Flat fields, single modal | `SimpleCrudPage` |
| Custom columns, export, 2FA, rich editor, tree, multi-modal | List + FormModal + `*.config.ts` |
## New admin CRUD modules
- **Backend:** use Dev → Code Generator or match `article_controller.go` / `controller.tpl`
- **Frontend:** generator produces `SimpleCrudPage` or List+FormModal+config — match output paradigm
- Do not hand-write legacy controller patterns (partial menu-style update in controller) for new modules
