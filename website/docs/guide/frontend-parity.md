# 双前端对齐清单（Vue / React）

本仓库提供两套管理端前端，对接**同一** `/api/admin`（及平台 `/api/platform`）契约。

## 主从约定

| 角色 | 目录 | 说明 |
|------|------|------|
| **主发货 UI（shipping）** | `html-react/`（React 19） | **默认上线前端**；文档截图、生产 Docker `BUILD_FRONTEND=1`（构建产物进 `public/admin`）、新功能优先落此端 |
| **对等参考实现（peer reference）** | `html/`（Vue 3） | 与 React 同仓库维护的 Vue 参考端；CI 独立 type-check / test / build，契约与权限对齐 |

新功能、修 Bug、改请求约定时：**先 React（`html-react/`），再跟进 Vue（`html/`）**。理想情况两端同 PR；若短期只能改一端，在 PR 中注明另一端跟进项。

## 改动时检查

- [ ] API 路径、query、body 字段与后端一致
- [ ] 权限 slug / 菜单路由两端都有对应页面或隐藏逻辑（React 先合入即可）
- [ ] `X-Tenant-ID` / `tenant_code` 公共资源 query（`applyTenantHeader` / `withTenantQuery`）行为一致
- [ ] i18n 文案键两端都有（或共用后端 `error_code`）
- [ ] 模块开关（`orders_enabled` / `payments_enabled` 等）前端默认与后端一致
- [ ] 能走代码生成器的 CRUD：优先生成 React（见 `CODE_GENERATOR_FRONTEND`，默认建议 `react,vue`）；Vue 模板可同批或随后补齐
- [ ] 新增纯工具函数：两端各加最小 vitest（参考 `src/utils/tenant.test.*`）

## 代码生成

后端 `app/services/templates/` 含 Vue 与 React 列表/表单模板。优先用生成器，再手工补差异（树形、仪表盘等）。环境变量 `CODE_GENERATOR_FRONTEND` 控制生成目标（`react` / `vue` / `react,vue` / `both`）。

## 相关文档

- [系统架构](/guide/architecture) — 整体架构
- [开源定位](/guide/opensource) — 开源定位与模块开关
- [测试指南](/guide/testing) — 测试与 CI
- [生产清单](/deploy/production) — Docker / SPA 构建说明
