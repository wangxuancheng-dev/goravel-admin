# 变更日志

本项目遵循 [语义化版本](https://semver.org/lang/zh-CN/) 规范。

## [Unreleased]

### 新增 / 优化
- ✨ React 管理端（`html-react/`）与 Vue 双端同发约定
- ✨ AI 实验室、代码生成器双端页面
- ✨ 分表查询服务、可插拔订单搜索驱动
- ✨ 支付方式 / 支付记录 / 网关服务拆分；订单筛选与导入下沉
- 📚 Docker Compose 本地一键（MySQL + Redis + seed）与 CI（Go 单测 + Vue/React build）
- 🔒 导出下载/进度/删除与附件读改删增加归属校验（所有者或 `admin.super_admin_id`）
- 🧪 CI 增加 `backend-integration`（MySQL + Redis + `tests/feature`）
- 📧 `send_email` 任务改为走框架 Mail；支付网关未实现能力返回稳定 `payment_gateway_not_implemented`
- 📚 `.env.production.example`、架构双前端与开源上线检查对齐
- 🏢 一户一库/Schema 多租户：默认 off 单库；开启后 Tenant→Jwt，认证与业务均在租户库；`tenant:seed`；导出/上传/缓存租户前缀
- 🏢 平台控制台（Landlord）：`platform_admins` + `/api/platform` + `/platform/login`；租户管理迁出租户后台；远程 DB 凭据；子域解析；`platform:admin` / `tenant:backup|restore`
- 🔒 远程租户建库打到目标 Host；`tenants.password` APP_KEY 加密；`skip_create`；mysqldump 密码走 defaults-extra-file
- ⚠️ 支付第三方回调 / 查询 / 退款仅为示例骨架，非正式支付中台（见 OPENSOURCE.md）

## [1.0.0] - 2024-12-25

### 新增

#### 核心功能
- ✨ 完整的后台管理系统
- ✨ JWT 认证 + Token 滑动过期
- ✨ RBAC 权限控制（角色-权限-菜单）
- ✨ 多 Token 管理和在线用户监控
- ✨ WebSocket 实时通知中心
- ✨ 中英文双语支持

#### 管理模块
- 👥 管理员管理（CRUD、密码重置）
- 🎭 角色管理（权限分配）
- 📋 菜单管理（动态菜单、图标）
- 🔐 权限管理（API 权限）
- 🏢 部门管理（树形结构）
- 📖 字典管理（配置项）
- 🚫 黑名单管理（IP/CIDR/范围）

#### 日志监控
- 📝 操作日志（自动记录）
- 🔑 登录日志
- ⚠️ 系统日志（TraceID 追踪）
- 📊 服务监控（CPU/内存/磁盘）

#### 文件管理
- 📁 附件上传（普通/分片）
- 📤 数据导出管理

### 技术特性

#### 后端
- 🔧 Goravel 框架
- 🔧 统一响应封装（response.go）
- 🔧 统一日志工具（Debug/Info/Warn/Error 分级）
- 🔧 TraceID 链路追踪
- 🔧 敏感字段过滤

#### 前端
- 🎨 Vue 3 + Composition API
- 🎨 Element Plus UI
- 🎨 VXE-Table 高性能表格
- 🎨 TypeScript 核心模块支持
- 🎨 Composables 高度复用

#### 测试
- ✅ 后端单元测试（tests/unit/）
- ✅ 后端集成测试（tests/feature/）
- ✅ 前端单元测试（Vitest）

#### 文档
- 📚 API 接口文档
- 📚 系统架构文档
- 📚 开发指南
- 📚 测试指南
- 📚 贡献指南

---

## 版本规范

- **主版本号 (Major)**: 不兼容的 API 变更
- **次版本号 (Minor)**: 向后兼容的功能新增
- **修订号 (Patch)**: 向后兼容的问题修复

### 变更类型

| 类型 | 说明 |
|------|------|
| ✨ 新增 | 新功能 |
| 🐛 修复 | Bug 修复 |
| 🔧 优化 | 性能或代码优化 |
| 📚 文档 | 文档更新 |
| ⚠️ 变更 | 不兼容的变更 |
| 🗑️ 移除 | 功能移除 |

---

