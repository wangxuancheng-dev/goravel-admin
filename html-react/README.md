# Goravel Admin (React)

Vue 版后台（`html/`）的 React 对照实现，对接同一套 Goravel Admin API。

<p align="center">
  <img src="../images/react.jpg" alt="React 后台管理界面" width="800">
  <p align="center">React 后台管理界面</p>
</p>

<p align="center">
  <img src="../images/ai-lab.png" alt="AI 实验室" width="800">
  <p align="center">AI 实验室（React）</p>
</p>

## 技术栈

- React 19 + TypeScript + Vite
- Ant Design 6
- Zustand / React Router 7 / axios / react-i18next

## 快速开始

```bash
cd html-react
cp .env.example .env
npm install
npm run dev
```

开发地址默认：`http://localhost:3008`  
后端默认：`http://localhost:3000`（与 `.env` 中 `VITE_API_BASE_URL` 一致）

> 若登录出现「网络连接失败」，多半是后端 CORS 未放行 React 端口。确认根目录 `.env` 含：
> `http://localhost:3008`，以及请求头 `Accept-Language`，然后**重启 Go 后端**。

> 生产环境托管 `dist` 时需做 SPA fallback（否则刷新子路由 404）。
> - Nginx：`try_files $uri $uri/ /index.html;`
> - Cloudflare Workers：使用本目录 `wrangler.toml` + `worker.js`，执行 `npx wrangler deploy`

默认账号与 Vue 版相同：`admin` / `admin123`

## 目录结构

```
src/
  api/          # 接口模块（createCRUDApi / extendApi）
  components/   # 通用 UI
  hooks/        # useListPage / usePermission 等
  i18n/         # 多语言
  layouts/      # 主布局、侧栏、多标签
  pages/        # 页面（菜单 component 字段映射到此）
  router/       # 静态路由 + 动态菜单路由
  stores/       # Zustand：user / app / tabs
  types/        # 类型与 API 契约
  utils/        # request / storage / normalize / tree ...
```

## 与 Vue 版对照

| Vue (`html/`) | React (`html-react/`) |
|---|---|
| Pinia | Zustand |
| Element Plus | Ant Design 6 |
| `useListPage` | `hooks/useListPage` |
| `utils/request.js` | `utils/request.ts` |
| `utils/apiFactory.js` | `utils/apiFactory.ts` |
| `views/**` | `pages/**` |

API 响应约定不变：`{ code, message, data, error_code, trace_id }`。

## 列表页范式

| 场景 | 写法 | 示例 |
|---|---|---|
| 字段简单、弹窗表单即可 | `SimpleCrudPage` | `position` / `dictionary` / `permission` / `blacklist` |
| 自定义列、导出、富文本、多弹窗等 | `*List.tsx` + `*FormModal.tsx` + `*.config.ts` | `article` / `admin` / `role` / `order` |

复杂页约定：
- `*.config.ts`：`initialSearchForm`、行类型、`transform*Row`、展示辅助函数
- `useListPage`：`fetchApi` 用 `ListFetchFn`；搜索用 `onSearchFormChange`（勿再 `as never`）
- 权限 slug 与后端一致（如 `admin.store`）

Agent 约定见 `.cursor/skills/goravel-admin-frontend-react/`。

## 当前已实现模块

- 登录 / 布局 / 动态菜单 / 多标签 / 权限按钮 / 通知铃铛（WS）
- 布局壳层：锁屏、时区切换、全屏、字号、顶栏/侧栏菜单、水印、主题色、列设置
- Dashboard（KPI + ECharts）、个人中心（头像 / 谷歌 2FA）
- 系统：管理员、角色（权限树）、权限、菜单、部门、岗位、在线管理员、字典、配置、导出、附件、黑名单
- 业务：用户（余额/重置密码/余额日志）、订单、支付方式、支付记录、文章（WangEditor）
- 通知：创建（Markdown / 富文本）、列表、详情
- 日志：操作 / 登录 / 系统（详情、批量删除、清理）+ 观测中心
- 监控：服务监控（SSE + ECharts）
- 开发：表单演示（含 WangEditor / Markdown）、代码生成器（Vue/React 双端，受 `CODE_GENERATOR_FRONTEND` 控制；菜单/权限安装见 [代码生成器文档](../website/docs/guide/code-generator.md)）
- **AI 实验室**（React）：文本对话、图片理解、图片生成、语音合成、语音转写；顶级菜单，配置 `AI_API_KEY` 后可用，按管理员账号限流（见 `.env.example`）
## 脚本

- `npm run dev` — 开发
- `npm run build` — 类型检查 + 生产构建
- `npm run type-check` — 仅类型检查
- `npm run preview` — 预览构建产物

更多说明见 [DEVELOPMENT.md](./DEVELOPMENT.md)。
