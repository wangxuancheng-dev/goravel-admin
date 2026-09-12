# Architecture

> This page mirrors the Chinese documentation for accuracy. Switch language to **简体中文**, or open the [Chinese version](/guide/architecture).

---

Goravel Admin 后台管理系统的整体架构。

---

## 系统概览

```
┌─────────────────────────────────────────────────────────────────┐
│                         客户端 (Browser)                         │
├───────────────────────────────┬─────────────────────────────────┤
│   React 19 SPA (html-react/)  │   Vue 3 SPA (html/)             │
│   Ant Design 6（shipping）    │   Element Plus + vxe-table      │
├───────────────────────────────┴─────────────────────────────────┤
│                         Nginx / CDN                              │
├─────────────────────────────────────────────────────────────────┤
│                      Goravel API Server                          │
│                    /api/admin（同一套契约）                        │
├─────────────────────────────────────────────────────────────────┤
│           MySQL / PostgreSQL          │         Redis            │
└─────────────────────────────────────────────────────────────────┘
```

两套前端**对接同一 Admin API**，约定新功能同发。**React（`html-react/`）为主发货 UI**，Vue（`html/`）为对等参考实现；对齐清单见 [双前端对齐](/guide/frontend-parity)。架构上后端一份，前端双实现。

可选多租户（默认 `TENANCY_DRIVER=off`）：设为 `database` 后为**一户一库/Schema**（连接级隔离，非行级 `tenant_id`）；`platform:install` 首启；平台控制台 `/api/platform`，租户业务在租户库。权威说明见 [多租户](/advanced/tenancy)。

---

## 技术架构

### 技术栈

| 层级 | 技术 | 版本 / 说明 |
|------|------|-------------|
| **后端框架** | Goravel | v1.18（以 `go.mod` 为准） |
| **编程语言** | Go | 1.25+（以 `go.mod` 为准） |
| **前端（React，主发货）** | React 19 + Ant Design 6 + Zustand + React Router 7 | 目录 `html-react/` |
| **前端（Vue，参考）** | Vue 3 + Element Plus + VXE-Table + Pinia | 目录 `html/` |
| **数据库** | MySQL / PostgreSQL | 8.0+ / 15+ |
| **缓存 / 队列** | Redis（可选 sync 队列本地跑） | 7.0+ |
| **认证** | JWT | - |

### 架构模式

```
┌──────────────────────────────┐  ┌──────────────────────────────┐
│   前端 React (html-react/)    │  │        前端 Vue (html/)       │
│  Pages / Components / Stores │  │  Views / Components / Store  │
│  request.js + apiFactory     │  │  request.ts + apiFactory      │
└──────────────┬───────────────┘  └──────────────┬───────────────┘
               │  /api/admin                      │
               └────────────────┬─────────────────┘
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                          后端 (Goravel)                          │
│  Controllers → Services → Models / Jobs / Events                 │
│  统一 response / BusinessError / 代码生成器模板                   │
├─────────────────────────────────────────────────────────────────┤
│  Database                          │  Redis（缓存/队列/锁）       │
└─────────────────────────────────────────────────────────────────┘
```

---

## 后端架构

### 目录结构

```
app/
├── console/                 # 命令行
│   └── commands/           # 自定义命令
├── http/
│   ├── controllers/        # 控制器
│   │   └── admin/         # 后台管理控制器
│   ├── middleware/         # 中间件
│   ├── requests/           # 请求验证
│   ├── helpers/            # 辅助函数
│   ├── response/           # 统一响应
│   └── trans/              # 翻译工具
├── models/                  # 数据模型
├── services/                # 业务逻辑
├── utils/                   # 工具函数
│   ├── logger/            # 日志工具
│   ├── traceid/           # 链路追踪
│   └── errorlog/          # 错误日志
├── events/                  # 事件
├── listeners/               # 事件监听器
├── jobs/                    # 队列任务
├── providers/               # 服务提供者
├── rules/                   # 自定义验证规则
└── websocket/               # WebSocket
    └── notifications/      # 通知推送
```

### 分层架构

#### 1. 控制器层 (Controllers)

负责接收请求、验证参数、调用服务、返回响应。

```go
// app/http/controllers/admin/admin_controller.go
func (a *AdminController) Index(ctx http.Context) http.Response {
    // 1. 获取查询参数
    // 2. 调用 Service 获取数据
    // 3. 返回统一响应
}
```

#### 2. 服务层 (Services)

封装业务逻辑，可被多个控制器复用。

```go
// app/services/admin_service.go
type AdminService struct{}

func (s *AdminService) GetAdminWithRoles(id uint) (*models.Admin, error) {
    // 业务逻辑处理
}
```

#### 3. 模型层 (Models)

定义数据结构和数据库映射。

```go
// app/models/admin.go
type Admin struct {
    orm.Model
    Username     string       `gorm:"size:50;uniqueIndex" json:"username"`
    Password     string       `gorm:"size:255" json:"-"`
    Roles        []*Role      `gorm:"many2many:admin_role" json:"roles,omitempty"`
}
```

#### 4. 中间件层 (Middleware)

处理认证、权限、日志等横切关注点。

```go
// 中间件执行顺序
Request → TraceID → JWT Auth → Permission → Controller → OperationLog → Response
```

**核心中间件：**

| 中间件 | 功能 |
|--------|------|
| `jwt.go` | JWT 认证 |
| `permission.go` | 权限验证 |
| `operation_log.go` | 操作日志记录 |
| `blacklist.go` | IP 黑名单检查 |
| `rate_limiter.go` | 请求限流 |
| `trace_id.go` | 链路追踪 |

### 统一响应

```go
// 成功响应
response.Success(ctx, data)

// 错误响应
response.Error(ctx, http.StatusBadRequest, "错误信息")

// 泛型查找
admin, resp := response.FindByID[models.Admin](ctx, id, nil)
```

---

## 前端架构

两套 SPA **共用后端契约**（`code` / `message` / `error_code` / `data.list`），目录与栈不同，业务模块应对齐。

| | React (`html-react/`，主发货) | Vue (`html/`，参考) |
|---|---|---|
| UI | Ant Design 6 | Element Plus + vxe-table |
| 状态 | Zustand | Pinia |
| 列表页 | `useListPage` + `SimpleCrudPage` 等 | `useListPage` / `useStandardListPage` |
| API | `apiFactory` + `request.ts` | `apiFactory` + `request.js` |
| i18n | `react-i18next`（同结构 locales） | `vue-i18n`（`locales/zh-CN|en-US.json`） |

**同发约定：** 新增或修改管理端功能时，优先落 React，再跟进 Vue；理想情况同一变更集交付（除非明确只改一端）。

### Vue 目录结构

```
html/src/
├── api/                     # API 请求
├── components/              # 通用组件
├── composables/             # 可复用逻辑
├── i18n/locales/            # 国际化
├── layouts/                 # 布局
├── router/                  # 路由
├── store/                   # Pinia
├── utils/request.js         # Axios 封装
└── views/                   # 页面
```

### React 目录结构

```
html-react/src/
├── api/                     # API 请求
├── components/              # 通用组件
├── hooks/                   # useListPage / usePermission 等
├── i18n/locales/            # 国际化
├── layouts/                 # 布局
├── pages/                   # 页面（List + FormModal + config）
├── stores/                  # Zustand
└── utils/request.ts         # Axios 封装
```

### Vue Composables / React Hooks

核心列表与权限逻辑两边各有封装，语义接近：

```typescript
// Vue: useCrud / usePermission（composables）
// React: useCrudActions / usePermission（hooks）
```

### 状态管理

**Vue（Pinia）**

```
userStore | appStore | tabsStore
```

**React（Zustand）**

```
user store | app store | tabs（若启用）
```

两边都保存 token、菜单树、权限码、语言与时区等，供请求头与按钮显隐使用。

### 资源归属

导出下载 / SSE / 删除，以及私有附件的读改删，校验 `admin_id`（所有者或配置 `admin.super_admin_id`）。公开附件可读，改删仍需归属。见 `ForbidUnlessOwnerOrSuper`。

---

## 数据库设计

### ER 图

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   admins    │─────│ admin_role  │─────│    roles    │
├─────────────┤     ├─────────────┤     ├─────────────┤
│ id          │     │ admin_id    │     │ id          │
│ username    │     │ role_id     │     │ name        │
│ password    │     └─────────────┘     │ slug        │
│ department_id│                        │ status      │
└──────┬──────┘                        └──────┬──────┘
       │                                       │
       │    ┌─────────────┐                   │
       │    │ departments │                   │
       └────┤             │     ┌─────────────┴─────────────┐
            │ id          │     │       role_permission      │
            │ name        │     ├───────────────────────────┤
            │ parent_id   │     │ role_id                   │
            └─────────────┘     │ permission_id             │
                               └─────────────┬─────────────┘
                                             │
                               ┌─────────────▼─────────────┐
                               │       permissions         │
                               ├───────────────────────────┤
                               │ id                        │
                               │ name                      │
                               │ slug                      │
                               │ method                    │
                               │ path                      │
                               │ menu_id                   │
                               └───────────────────────────┘
```

### 核心表结构

| 表名 | 说明 |
|------|------|
| `admins` | 管理员 |
| `roles` | 角色 |
| `permissions` | 权限 |
| `menus` | 菜单 |
| `departments` | 部门 |
| `dictionaries` | 字典 |
| `blacklists` | 黑名单 |
| `operation_logs` | 操作日志 |
| `login_logs` | 登录日志 |
| `system_logs` | 系统日志 |
| `personal_access_tokens` | Token 管理 |
| `notifications` | 通知 |
| `attachments` | 附件 |
| `exports` | 导出任务 |

---

## 安全架构

### 认证流程

```
┌─────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│  Login  │───▶│ Validate    │───▶│ Generate    │───▶│   Return    │
│ Request │    │ Credentials │    │   JWT       │    │   Token     │
└─────────┘    └─────────────┘    └─────────────┘    └─────────────┘
                     │
                     ▼
              ┌─────────────┐
              │ Rate Limit  │
              │   Check     │
              └─────────────┘
```

### 权限验证流程

```
Request → JWT Middleware → Permission Middleware → Controller
              │                    │
              ▼                    ▼
        验证 Token           检查权限
              │                    │
              ▼                    ▼
        解析用户信息         匹配路由权限
              │                    │
              ▼                    ▼
        写入 Context         允许/拒绝
```

### 安全特性

| 特性 | 实现方式 |
|------|----------|
| **认证** | JWT Token + 滑动过期 |
| **授权** | RBAC 权限模型 |
| **限流** | Token Bucket 算法 |
| **黑名单** | IP/Token 双重黑名单 |
| **日志** | 操作/登录日志全记录 |
| **追踪** | TraceID 链路追踪 |
| **敏感字段** | 密码等字段 `json:"-"` |

---

## 部署架构

### 单机部署

```
┌────────────────────────────────────────┐
│              Server                     │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐ │
│  │  Nginx  │──│ Goravel │──│  MySQL  │ │
│  └─────────┘  └─────────┘  └─────────┘ │
│                    │                    │
│               ┌────▼────┐              │
│               │  Redis  │              │
│               └─────────┘              │
└────────────────────────────────────────┘
```

### 高可用部署

```
                    ┌─────────────┐
                    │   Client    │
                    └──────┬──────┘
                           │
                    ┌──────▼──────┐
                    │     CDN     │
                    └──────┬──────┘
                           │
                    ┌──────▼──────┐
                    │ Load Balancer│
                    └──────┬──────┘
           ┌───────────────┼───────────────┐
           │               │               │
    ┌──────▼──────┐ ┌──────▼──────┐ ┌──────▼──────┐
    │   Goravel   │ │   Goravel   │ │   Goravel   │
    │   Server 1  │ │   Server 2  │ │   Server 3  │
    └──────┬──────┘ └──────┬──────┘ └──────┬──────┘
           │               │               │
           └───────────────┼───────────────┘
                           │
              ┌────────────┼────────────┐
              │                         │
       ┌──────▼──────┐          ┌──────▼──────┐
       │ MySQL Master│──────────│ Redis Cluster│
       └──────┬──────┘          └──────────────┘
              │
       ┌──────▼──────┐
       │ MySQL Slave │
       └─────────────┘
```

### Docker 部署

```yaml
# docker-compose.yml
version: '3.8'
services:
  app:
    build: .
    ports:
      - "3000:3000"
    depends_on:
      - mysql
      - redis

  mysql:
    image: mysql:8.0
    volumes:
      - mysql_data:/var/lib/mysql

  redis:
    image: redis:7-alpine
    volumes:
      - redis_data:/data
```

---

## 性能优化

### 后端优化

| 优化项 | 实现 |
|--------|------|
| 数据库连接池 | 配置合理的连接数 |
| 查询优化 | 预加载关联、索引优化 |
| 缓存策略 | Redis 缓存热点数据 |
| 日志分级 | Debug/Info 分离 |

### 前端优化

| 优化项 | 实现 |
|--------|------|
| 虚拟滚动 / 大数据表 | Vue：VXE-Table；React：Ant Design Table + 分页 |
| 代码分割 | 路由懒加载 |
| 状态管理 | Pinia / Zustand |
| 防抖节流 | 列表搜索 hooks/composables |

---

## 扩展指南

### 添加新模块

1. **后端**
   - Model / Service / Controller / Request（优先对齐代码生成器模板）
   - 注册路由: `routes/admin.go`
   - 菜单与权限 seed / 安装器

2. **前端（Vue + React 同发）**
   - Vue: `html/src/api/` + `html/src/views/`
   - React: `html-react/src/api/` + `html-react/src/pages/`
   - 两边 i18n locales 补齐本模块用到的 key

### 添加新权限

1. 数据库添加权限记录 (`permissions` 表)
2. 关联到菜单 (`menu_id`)
3. 分配给角色 (`role_permission` 表)

---

## 参考资料

- [Goravel 官方文档](https://www.goravel.dev)
- [Vue 3 文档](https://vuejs.org)
- [Element Plus 文档](https://element-plus.org)
- [React 文档](https://react.dev)
- [Ant Design 文档](https://ant.design)
- [前端 skill：Vue](../.cursor/skills/goravel-admin-frontend/SKILL.md) / [React](../.cursor/skills/goravel-admin-frontend-react/SKILL.md)

