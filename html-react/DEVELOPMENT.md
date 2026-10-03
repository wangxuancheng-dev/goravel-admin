# React 前端开发指南

本文档说明 `html-react/` 的架构约定，设计目标是与 `html/`（Vue）保持同一套业务契约与分层，同时在 React 侧做到可维护的封装。

## 技术栈

| 技术 | 说明 |
|------|------|
| React 19 | UI |
| Vite | 构建 |
| Ant Design 6 | 组件库 |
| Zustand | 全局状态（对标 Pinia） |
| React Router 7 | 路由 + 动态菜单 |
| axios | HTTP（拦截器与 Vue 版行为对齐） |
| react-i18next | 国际化 |

## 分层约定

1. **pages/**：页面只负责拼装 UI 与调用 hooks/api，不直接拼 axios。
2. **api/**：所有接口模块；CRUD 用 `createCRUDApi`，扩展用 `extendApi`。
3. **hooks/**：列表页用 `useListPage`；权限用 `usePermission`。
4. **stores/**：`user`（登录/权限/菜单）、`app`（主题/语言相关外观）、`tabs`（多标签）。
5. **utils/request.ts**：统一处理 token、语言、时区、401/403、业务 `code !== 200`。

## 后端契约（禁止改形状）

成功：

```json
{ "code": 200, "message": "...", "data": {}, "trace_id": "..." }
```

失败：

```json
{ "code": 401, "message": "...", "error_code": "unauthorized", "trace_id": "..." }
```

错误展示规则：

- 非登录接口：拦截器已 toast 的错误带 `err.__handled === true`，页面不要再弹一次。
- `/login`、`/logout`：交给页面本地处理（验证码、谷歌码等）。

## 新增 CRUD 模块

```ts
// src/api/widget.ts
import { createCRUDApi } from '@/utils/apiFactory'
import { normalizeListResponse } from '@/utils/normalize'

const widgetApi = createCRUDApi('widgets')

export async function getWidgetList(params?: Record<string, unknown>) {
  return normalizeListResponse(await widgetApi.list(params))
}

export const getWidgetDetail = widgetApi.detail
export const createWidget = widgetApi.create
export const updateWidget = widgetApi.update
export const deleteWidget = widgetApi.delete
```

### 选哪种页面写法

| 场景 | 写法 |
|------|------|
| 字段简单、单弹窗即可 | `SimpleCrudPage`（见 `pages/position/PositionList.tsx`） |
| 自定义列 / 导出 / 富文本 / 多弹窗 | `*List.tsx` + `*FormModal.tsx` + `*.config.ts`（见 `pages/article/`、`pages/admin/`） |

复杂列表示例：

```tsx
const { tableData, onSearchFormChange, handleSearch, handleReset, ... } = useListPage<Row, SearchForm>({
  fetchApi: getWidgetList, // API module should call normalizeListResponse
  initialSearchForm,
  transformData: transformWidgetRow,
})
```

- 搜索表单用 `onSearchFormChange`，不要 `as never`
- 列表字段归一化只放在 API 的 `normalizeListResponse`，不要在 `useListPage` 再做一遍
- `*.config.ts` 放 `initialSearchForm`、行类型、`transform*Row`
- 页面文件放到 `src/pages/**`，菜单 `component` 填 `widget/WidgetList` 即可被动态路由加载

Agent 约定：`.cursor/skills/goravel-admin-frontend-react/`。

## 权限按钮

```tsx
import PermissionButton from '@/components/PermissionButton'

<PermissionButton permission="admin.store" type="primary">
  添加
</PermissionButton>
```

`usePermission().getButtonState(slug)` 返回 `{ show, disabled }`，逻辑与 Vue 版一致（含超级管理员与“无权限仍显示按钮”配置）。

## 环境变量

```env
VITE_API_BASE_URL=http://127.0.0.1:3000
VITE_API_PREFIX=/api/admin
```

开发端口：`3008`（避免与 Vue 的 `3007` 冲突）。WebSocket `/ws` 与公开资源 `/api/admin/public` 已在 Vite 代理。

## 质量要求

- TypeScript strict；公共类型放 `src/types`。
- 列表请求参数统一经 `buildSearchParams`（空值过滤）。
- 实体字段读写优先 `entityField` / `normalizeListResponse`，兼容 snake_case / PascalCase。
- 不要在页面里复制一套 401/403 处理。
