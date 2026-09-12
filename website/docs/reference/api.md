# API 接口文档

本文档详细描述了 Goravel Admin 后台管理系统的所有 API 接口。

## 基础信息

- **Base URL**: `http://localhost:3000/api/admin`
- **认证方式**: JWT Bearer Token
- **请求格式**: JSON
- **响应格式**: JSON

## 契约同步说明（2026-04）

为避免文档与实现漂移，当前以 `routes/admin.go` 和 Swagger 产物为准。以下是已对齐的关键接口：

- 认证与会话：
  - `POST /login`
  - `GET /login/captcha`
  - `GET /info`
  - `POST /logout`
  - `GET /heartbeat`
- 通知中心：
  - `GET /notifications`
  - `GET /notifications/unread-count`
  - `GET /notifications/recent`
  - `POST /notifications/ws-ticket`（获取一次性 WS ticket）
  - `POST /notifications/{id}/read`
  - `POST /notifications/read-all`
- WebSocket：
  - `GET /ws/admin/notifications`（非 `/api/admin/*` 前缀）
  - 仅支持 `ticket` 参数进行鉴权，不再支持 `token` query
- 日志：
  - `GET /operation-logs`
  - `GET /login-logs`
  - `GET /system-logs`
- 监控：
  - `GET /monitor/system-info`
  - `GET /monitor/system-info/stream`

> 说明：本文后续部分若与上述不一致，请以上述“契约同步说明”和实际路由为准。

## 接口速查表（联调用）

### 认证与会话

- `POST /login`
- `GET /login/captcha`
- `GET /info`
- `POST /logout`
- `GET /heartbeat`
- `PUT /profile`
- `PUT /password`

### 通知与实时

- `GET /notifications`
- `GET /notifications/unread-count`
- `GET /notifications/recent`
- `POST /notifications/ws-ticket`
- `POST /notifications/{id}/read`
- `POST /notifications/read-all`
- `POST /notifications`
- `GET /ws/admin/notifications?ticket={ticket}`

### 日志与监控

- `GET /operation-logs`
- `GET /login-logs`
- `GET /system-logs`
- `GET /dashboard/count`
- `GET /dashboard/user-access-source`
- `GET /dashboard/weekly-user-activity`
- `GET /dashboard/monthly-sales`
- `GET /dashboard/recent-activities`
- `GET /dashboard/stream`
- `GET /monitor/system-info`
- `GET /monitor/system-info/stream`

### 用户与订单支付

- `GET /users`
- `POST /users/{id}/update-balance`
- `PUT /users/{id}/password`
- `POST /users/export`
- `GET /user-balance-logs`
- `POST /user-balance-logs`
- `GET /user-balance-logs/statistics`
- `GET /orders`
- `POST /orders/export`
- `POST /orders/import`
- `GET /orders/export/status/{id}`
- `GET /payments`
- `GET /payments/{id}`
- `POST /payments/export`
- `GET /payments/export/status/{id}`

### 附件与导出

- `GET /attachments`
- `POST /attachments/upload`
- `POST /attachments/chunk`（`action=init|upload|merge`）
- `GET /attachments/chunk`（`action=progress`）
- `GET /attachments/{id}/preview`
- `GET /attachments/{id}/download`
- `PUT /attachments/{id}/display-name`
- `DELETE /attachments/{id}`
- `POST /attachments/batch-delete`
- `GET /exports`
- `GET /exports/{id}/download`
- `DELETE /exports/{id}`
- `POST /exports/batch-delete`

## 通用响应格式

### 成功响应

```json
{
  "code": 200,
  "message": "操作成功",
  "data": {}
}
```

### 分页响应

```json
{
  "code": 200,
  "message": "操作成功",
  "data": {
    "list": [],
    "page": 1,
    "page_size": 10,
    "total": 100
  }
}
```

### 错误响应

```json
{
  "code": 400,
  "message": "错误信息",
  "data": null
}
```

### 状态码说明

| 状态码 | 说明 |
|--------|------|
| 200 | 成功 |
| 400 | 请求参数错误 |
| 401 | 未认证或 Token 过期 |
| 403 | 无权限访问 |
| 404 | 资源不存在 |
| 422 | 验证失败 |
| 500 | 服务器内部错误 |

---

## 认证接口

### 登录

```http
POST /login
```

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| username | string | 是 | 用户名 |
| password | string | 是 | 密码 |
| captcha_id | string | 否 | 验证码ID |
| captcha_code | string | 否 | 验证码 |

**请求示例：**

```json
{
  "username": "admin",
  "password": "admin123"
}
```

**响应示例：**

```json
{
  "code": 200,
  "message": "登录成功",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_at": "2024-01-01T12:00:00Z"
  }
}
```

### 退出登录

```http
POST /logout
Authorization: Bearer {token}
```

**响应示例：**

```json
{
  "code": 200,
  "message": "退出成功",
  "data": null
}
```

### 获取验证码

```http
GET /login/captcha
```

**响应示例：**

```json
{
  "code": 200,
  "data": {
    "captcha_id": "abc123",
    "captcha_image": "data:image/png;base64,..."
  }
}
```

---

## 管理员信息

### 获取当前用户信息

```http
GET /info
Authorization: Bearer {token}
```

**响应示例：**

```json
{
  "code": 200,
  "data": {
    "id": 1,
    "username": "admin",
    "nickname": "超级管理员",
    "email": "admin@example.com",
    "avatar": "",
    "department_id": 1,
    "department": {
      "id": 1,
      "name": "总部"
    },
    "roles": [
      {
        "id": 1,
        "name": "超级管理员",
        "slug": "super-admin"
      }
    ],
    "permissions": ["admin.index", "admin.store", "..."],
    "menus": []
  }
}
```

### 修改密码

```http
PUT /password
Authorization: Bearer {token}
```

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| old_password | string | 是 | 原密码 |
| password | string | 是 | 新密码 |
| password_confirmation | string | 是 | 确认新密码 |

---

## 管理员管理

### 获取管理员列表

```http
GET /admins
Authorization: Bearer {token}
```

**查询参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| page | int | 页码，默认 1 |
| page_size | int | 每页条数，默认 10 |
| username | string | 用户名（模糊搜索） |
| nickname | string | 昵称（模糊搜索） |
| status | int | 状态：0-禁用，1-启用 |
| department_id | int | 部门ID |
| order_by | string | 排序，如 `id:desc` |

**响应示例：**

```json
{
  "code": 200,
  "data": {
    "list": [
      {
        "id": 1,
        "username": "admin",
        "nickname": "超级管理员",
        "email": "admin@example.com",
        "phone": "",
        "avatar": "",
        "status": 1,
        "department_id": 1,
        "department": {"id": 1, "name": "总部"},
        "roles": [{"id": 1, "name": "超级管理员"}],
        "created_at": "2024-01-01T00:00:00Z",
        "last_login_at": "2024-01-01T12:00:00Z"
      }
    ],
    "page": 1,
    "page_size": 10,
    "total": 1
  }
}
```

### 创建管理员

```http
POST /admins
Authorization: Bearer {token}
```

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| username | string | 是 | 用户名（唯一） |
| password | string | 是 | 密码 |
| nickname | string | 是 | 昵称 |
| email | string | 否 | 邮箱 |
| phone | string | 否 | 手机号 |
| avatar | string | 否 | 头像URL |
| department_id | int | 否 | 部门ID |
| role_ids | []int | 否 | 角色ID数组 |
| status | int | 否 | 状态，默认 1 |

### 获取管理员详情

```http
GET /admins/{id}
Authorization: Bearer {token}
```

### 更新管理员

```http
PUT /admins/{id}
Authorization: Bearer {token}
```

### 删除管理员

```http
DELETE /admins/{id}
Authorization: Bearer {token}
```

### 重置密码

```http
PUT /admins/{id}/password
Authorization: Bearer {token}
```

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| password | string | 是 | 新密码 |

---

## 角色管理

### 获取角色列表

```http
GET /roles
Authorization: Bearer {token}
```

**查询参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| page | int | 页码 |
| page_size | int | 每页条数 |
| name | string | 角色名称（模糊搜索） |
| status | int | 状态 |

### 创建角色

```http
POST /roles
Authorization: Bearer {token}
```

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 角色名称 |
| slug | string | 是 | 角色标识（唯一） |
| description | string | 否 | 描述 |
| permission_ids | []int | 否 | 权限ID数组 |
| menu_ids | []int | 否 | 菜单ID数组 |
| status | int | 否 | 状态 |
| sort | int | 否 | 排序 |

### 获取角色详情

```http
GET /roles/{id}
Authorization: Bearer {token}
```

### 更新角色

```http
PUT /roles/{id}
Authorization: Bearer {token}
```

### 删除角色

```http
DELETE /roles/{id}
Authorization: Bearer {token}
```

---

## 菜单管理

### 获取菜单列表（树形）

```http
GET /menus
Authorization: Bearer {token}
```

**响应示例：**

```json
{
  "code": 200,
  "data": {
    "list": [
      {
        "id": 1,
        "parent_id": 0,
        "title": "系统管理",
        "slug": "system",
        "icon": "Setting",
        "path": "/system",
        "component": "",
        "type": 1,
        "status": 1,
        "sort": 1,
        "children": [
          {
            "id": 2,
            "parent_id": 1,
            "title": "管理员管理",
            "slug": "admin",
            "path": "/system/admin",
            "component": "admin/AdminList",
            "type": 2,
            "status": 1
          }
        ]
      }
    ]
  }
}
```

### 创建菜单

```http
POST /menus
Authorization: Bearer {token}
```

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| parent_id | int | 否 | 父级ID，默认 0 |
| title | string | 是 | 菜单标题 |
| slug | string | 是 | 菜单标识 |
| icon | string | 否 | 图标名称 |
| path | string | 否 | 路由路径 |
| component | string | 否 | 组件路径 |
| type | int | 是 | 类型：1-目录，2-菜单，3-按钮 |
| status | int | 否 | 状态 |
| sort | int | 否 | 排序 |

### 获取菜单详情

```http
GET /menus/{id}
Authorization: Bearer {token}
```

### 更新菜单

```http
PUT /menus/{id}
Authorization: Bearer {token}
```

### 删除菜单

```http
DELETE /menus/{id}
Authorization: Bearer {token}
```

---

## 权限管理

### 获取权限列表

```http
GET /permissions
Authorization: Bearer {token}
```

### 创建权限

```http
POST /permissions
Authorization: Bearer {token}
```

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 权限名称 |
| slug | string | 是 | 权限标识（唯一） |
| method | string | 是 | HTTP 方法（GET/POST/PUT/DELETE） |
| path | string | 是 | API 路径 |
| menu_id | int | 否 | 关联菜单ID |
| status | int | 否 | 状态 |

### 更新权限

```http
PUT /permissions/{id}
Authorization: Bearer {token}
```

### 删除权限

```http
DELETE /permissions/{id}
Authorization: Bearer {token}
```

---

## 部门管理

### 获取部门列表（树形）

```http
GET /departments
Authorization: Bearer {token}
```

### 创建部门

```http
POST /departments
Authorization: Bearer {token}
```

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 部门名称 |
| parent_id | int | 否 | 父部门ID |
| leader | string | 否 | 负责人 |
| phone | string | 否 | 联系电话 |
| email | string | 否 | 邮箱 |
| status | int | 否 | 状态 |
| sort | int | 否 | 排序 |

### 更新部门

```http
PUT /departments/{id}
Authorization: Bearer {token}
```

### 删除部门

```http
DELETE /departments/{id}
Authorization: Bearer {token}
```

---

## 字典管理

### 获取字典列表

```http
GET /dictionaries
Authorization: Bearer {token}
```

### 创建字典

```http
POST /dictionaries
Authorization: Bearer {token}
```

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| parent_id | int | 否 | 父级ID（0为字典类型） |
| name | string | 是 | 名称 |
| code | string | 是 | 编码 |
| value | string | 否 | 值 |
| description | string | 否 | 描述 |
| status | int | 否 | 状态 |
| sort | int | 否 | 排序 |

### 更新字典

```http
PUT /dictionaries/{id}
Authorization: Bearer {token}
```

### 删除字典

```http
DELETE /dictionaries/{id}
Authorization: Bearer {token}
```

---

## 黑名单管理

### 获取黑名单列表

```http
GET /blacklists
Authorization: Bearer {token}
```

### 创建黑名单

```http
POST /blacklists
Authorization: Bearer {token}
```

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| ip | string | 是 | IP地址（支持单IP、CIDR、范围） |
| remark | string | 否 | 备注 |
| status | int | 否 | 状态 |

**IP 格式示例：**

- 单个 IP: `192.168.1.100`
- CIDR 格式: `192.168.1.0/24`
- IP 范围: `192.168.1.1-192.168.1.100`

### 更新黑名单

```http
PUT /blacklists/{id}
Authorization: Bearer {token}
```

### 删除黑名单

```http
DELETE /blacklists/{id}
Authorization: Bearer {token}
```

### 批量删除黑名单

```http
DELETE /blacklists/batch
Authorization: Bearer {token}
```

**请求参数：**

```json
{
  "ids": [1, 2, 3]
}
```

---

## 日志管理

### 操作日志

```http
GET /operation-logs
Authorization: Bearer {token}
```

**查询参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| admin_id | int | 管理员ID |
| module | string | 模块 |
| action | string | 操作 |
| start_date | string | 开始日期 |
| end_date | string | 结束日期 |

### 登录日志

```http
GET /login-logs
Authorization: Bearer {token}
```

**查询参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| admin_id | int | 管理员ID |
| ip | string | IP地址 |
| status | int | 状态：0-失败，1-成功 |
| start_date | string | 开始日期 |
| end_date | string | 结束日期 |

### 系统日志

```http
GET /system-logs
Authorization: Bearer {token}
```

**查询参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| level | string | 日志级别（error/warning/info/debug） |
| trace_id | string | 追踪ID |
| start_date | string | 开始日期 |
| end_date | string | 结束日期 |

---

## 仪表盘

### 获取统计数据

```http
GET /dashboard/count
Authorization: Bearer {token}
```

**响应示例：**

```json
{
  "code": 200,
  "data": {
    "admin_count": 10,
    "role_count": 5,
    "menu_count": 30,
    "login_count_today": 15,
    "operation_count_today": 100,
    "recent_logins": [],
    "recent_operations": []
  }
}
```

---

## 在线管理员

### 获取在线管理员列表

```http
GET /online-admins
Authorization: Bearer {token}
```

### 强制下线

```http
DELETE /online-admins/{id}
Authorization: Bearer {token}
```

### 批量强制下线

```http
POST /online-admins/batch-kick-out
Authorization: Bearer {token}
```

**请求参数：**

```json
{
  "token_ids": "1,2,3"
}
```

---

## 服务监控

### 获取系统信息

```http
GET /monitor/system-info
Authorization: Bearer {token}
```

### 实时系统信息（SSE）

```http
GET /monitor/system-info/stream
Authorization: Bearer {token}
```

**响应示例：**

```json
{
  "code": 200,
  "data": {
    "cpu": {
      "cores": 8,
      "usage": 25.5
    },
    "memory": {
      "total": 16384,
      "used": 8192,
      "usage": 50.0
    },
    "disk": {
      "total": 512000,
      "used": 256000,
      "usage": 50.0
    },
    "go": {
      "version": "go1.21",
      "goroutines": 50,
      "gc_pause": "1.2ms"
    }
  }
}
```

---

## 文件上传

### 上传文件

```http
POST /attachments/upload
Authorization: Bearer {token}
Content-Type: multipart/form-data
```

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| file | file | 是 | 文件 |
| type | string | 否 | 类型（image/document/video） |

### 分片上传

```http
POST /attachments/chunk
Authorization: Bearer {token}
Content-Type: multipart/form-data
```

> 说明：当前分片上传采用统一接口，通过 `action` 参数区分：
> - `action=init`：初始化
> - `action=upload`：上传分片
> - `action=merge`：合并分片
> - `action=progress`：查询进度（通常 GET）

**请求参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| file | file | 是 | 分片文件 |
| chunk_id | string | 是 | 分片标识 |
| chunk_index | int | 是 | 分片索引 |
| total_chunks | int | 是 | 总分片数 |
| filename | string | 是 | 原始文件名 |

### 合并分片

```http
POST /attachments/chunk
Authorization: Bearer {token}
Content-Type: multipart/form-data
```

**请求参数：**

```json
{
  "chunk_id": "abc123",
  "filename": "large-file.zip",
  "total_chunks": 10
}
```

---

## 通知中心

### 获取 WebSocket Ticket

```http
POST /notifications/ws-ticket
Authorization: Bearer {token}
```

**响应示例：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "ticket": "01jvxxxxxxxxxxxxxxxxxxxxxx",
    "expires_in": 60
  }
}
```

### WebSocket 连接

```
ws://localhost:3000/ws/admin/notifications?ticket={ticket}
```

**消息格式：**

```json
{
  "type": "notification",
  "data": {
    "id": 1,
    "title": "新通知",
    "content": "通知内容",
    "created_at": "2024-01-01T12:00:00Z"
  }
}
```

### 获取通知列表

```http
GET /notifications
Authorization: Bearer {token}
```

### 标记已读

```http
POST /notifications/{id}/read
Authorization: Bearer {token}
```

### 全部标记已读

```http
POST /notifications/read-all
Authorization: Bearer {token}
```

---

## 错误码说明

错误码定义在 `app/errors/codes.go`，格式为 `XXYYYY`：
- `XX`: 模块 (10-认证, 20-权限, 30-验证, 40-业务, 50-系统)
- `YYYY`: 具体错误

### 认证模块 (10xxx)

| 错误码 | 说明 |
|--------|------|
| 10001 | 用户名或密码错误 |
| 10002 | 账号已被禁用 |
| 10003 | Token 已过期 |
| 10004 | Token 已被撤销 |
| 10005 | 验证码错误 |
| 10006 | 登录尝试次数超限 |
| 10007 | Token 无效 |

### 权限模块 (20xxx)

| 错误码 | 说明 |
|--------|------|
| 20001 | 无权限访问 |
| 20002 | 资源不存在 |
| 20003 | 访问被拒绝 |

### 验证模块 (30xxx)

| 错误码 | 说明 |
|--------|------|
| 30001 | 参数验证失败 |
| 30002 | 数据已存在 |
| 30003 | 数据不存在 |
| 30004 | 格式无效 |

### 业务模块 (40xxx)

| 错误码 | 说明 |
|--------|------|
| 40001 | 操作失败 |
| 40002 | 删除失败 |
| 40003 | 更新失败 |
| 40004 | 创建失败 |
| 40005 | 上传失败 |
| 40006 | 导出失败 |

### 系统模块 (50xxx)

| 错误码 | 说明 |
|--------|------|
| 50001 | 服务器内部错误 |
| 50002 | 数据库错误 |
| 50003 | 缓存错误 |
| 50004 | 队列错误 |
| 50005 | 第三方服务错误 |

---

## 附录

### 认证头格式

```http
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
```

### 分页参数

所有列表接口支持以下分页参数：

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| page | int | 1 | 页码 |
| page_size | int | 10 | 每页条数（最大 100） |
| order_by | string | id:desc | 排序（字段:方向） |

### 时间格式

所有时间字段使用 ISO 8601 格式：`2024-01-01T12:00:00Z`

### 时区请求头（响应时间字段转换）

- 后端审计时间（如 `created_at` / `updated_at`）以统一基准存储与返回。
- 仅当请求显式携带时区信息时，后端才会对响应中的时间字段做展示层转换。
- 支持以下时区输入（优先级从高到低）：`timezone` 参数、`X-Timezone` 请求头、`Timezone` 请求头。
- 未携带时区时，不执行响应时间字段转换，返回后端原值。
- 默认转换字段白名单：`created_at`、`updated_at`、`deleted_at`（兼容驼峰命名）。
- 可通过配置 `app.response_time_fields`（逗号分隔）扩展需要转换的字段。

### 多语言支持

通过请求头指定语言：

```http
Accept-Language: zh-CN
```

支持的语言：
- `zh-CN` - 简体中文
- `en-US` - English

