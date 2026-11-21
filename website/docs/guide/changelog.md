# 变更日志

本项目遵循 [语义化版本](https://semver.org/lang/zh-CN/) 规范。

## [Unreleased]

### 新增 / 优化
- 📚 文档驾驭复杂度：开源定位增加阅读路径与 A/B/C 能力分档；生产清单补充「中小部署暂时不需要懂」；React 为默认前端、Vue 对齐（见双前端对齐）
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
- 🛠️ `platform:install` 首启；HTTP 开户禁止同步 migrate；平台控制台支持单户异步 migrate/seed/backup（`tenant_ops`）
- 🔒 多租户防护：多租户开启时 `OrmQuery(nil)` 直接报错（不再悄悄落到平台库），平台查询改为显式的 `PlatformQuery()`；租户连接拒绝读写平台表（`models.PlatformModel`）；后台协程统一用 `tenancyctx.Go`（保留租户、兜底 panic）；新增可选的 `TENANCY_DB_DEDICATED_USER`，为每个租户自动创建只能访问自己库的 DB 账号
- 🧪 `scripts/tenantlint` 取代 `check-tenant-orm.sh` / `check-tenant-cache.sh`，按调用点检查（例外须写明原因）；新增全接口跨租户扫描与租户解析安全测试
- 🐛 关闭租户连接时，进行中查询的遥测回调不再因 nil 指针 panic
- 🔒 永久删除租户（含平台异步清理与定时 `tenant:cleanup-deleted`）一律删除其库、对象存储、备份与搜索索引后才释放 code；开户不再接管同名已存在的库（`tenant_storage_exists`）；开户失败物理删除并清理本次所建，code 立即可重用
- 🔒 多租户开启时，未绑定租户的 ctx（不只 nil）调用 `OrmQuery` 一律报 `ErrNoTenantContext`；队列统计命令改为显式读平台库，`token:create` 需 `--tenant` 并在租户库建 token
- 🩺 租户库或平台库暂时不可用时返回 503 + `Retry-After`（原为 500，平台库故障还会被报成 404 租户不存在）；生产环境未配置 `TENANCY_BASE_DOMAIN` 时启动告警
- 🔒 `TRUSTED_FORWARDED_HOST_ONLY=true`（可选，默认关闭以兼容旧部署）：`X-Forwarded-Host` 只采信 `TRUSTED_PROXIES` 内的对端，防止伪造 Host 选租户；新增 `TENANCY_MYSQL_TLS`，租户 MySQL 连接与备份可走 TLS
- 🛠️ 新增 `tenant:orphans`：列出（`--drop` 确认后删除）没有任何租户引用的残留租户库 / schema，升级后执行一次
- 🔒 租户无自有账号时回落平台账号，每次连接都受 `TENANCY_ALLOW_PLATFORM_DB_CREDENTIALS` 约束（含备份）；`platform.` 主机不再接受客户端指定租户
- 📚 多租户文档拆分：[业务开发须知](/advanced/tenancy-dev)、[规模与推荐配置](/advanced/tenancy-scaling)
- 🐛 OIDC SSO 回调此前读取 state 的方式有误，所有登录都返回 `oidc_invalid_state`，现已可用（state 一次性消费）
- 🔒 OIDC 只接受 IdP 标记 `email_verified=true` 的邮箱（防止用未验证邮箱冒用同邮箱管理员）；IdP 不下发该声明时可设 `OIDC_ALLOW_UNVERIFIED_EMAIL=true`（仅进程级）。**升级注意**：使用 Microsoft Entra ID 等默认不下发该声明的 IdP，需先在 IdP 侧配置或设置该开关
- 🔒 `OIDC_AUTO_PROVISION=true` 新建的管理员此前因 gorm 默认值实际为启用状态并直接登录，现确为禁用
- 🐛 创建订单的防重复提交锁此前从未生效（同一 `request_id` 并发提交会生成多张订单），改为原子加锁
- 🔒 模型状态 / 开关字段去掉 gorm `default:1/true`：gorm 会在 INSERT 前把 0/false 换成默认值，导致新建时选的「禁用 / 私有 / 不公开 / 失败」全部存成了相反值。受影响：管理员、会员、角色、权限、菜单、部门、岗位、字典、附件分类、黑白名单的状态；**附件的私有（文档类附件默认私有，实际全部公开，可免登录经 `/api/public/files/{id}` 访问）**；操作日志 / 平台操作日志的失败状态；套餐不公开、功能状态、entitlement 覆盖 `pinned=false`；租户自定义存储关闭 SSL；货币小数位 0（如日元）。新增 `app/models/default_tag_test.go` 防止再犯。**升级注意**：已入库的数据不会自动修正，建议检查附件的 `is_public`，以及新建时本应禁用的管理员 / 会员

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

